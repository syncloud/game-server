package installer

import (
	cp "github.com/otiai10/copy"
	"github.com/syncloud/golib/config"
	"github.com/syncloud/golib/linux"
	"github.com/syncloud/golib/platform"
	"go.uber.org/zap"
	"os"
	"path"
)

const (
	App       = "game-server"
	AppDir    = "/snap/game-server/current"
	DataDir   = "/var/snap/game-server/current"
	CommonDir = "/var/snap/game-server/common"
)

type Variables struct {
	AuthUrl         string
	AuthLocalSocket string
}

type Installer struct {
	newVersionFile     string
	currentVersionFile string
	configDir          string
	platformClient     *platform.Client
	installFile        string
	executor           *Executor
	logger             *zap.Logger
}

func New(logger *zap.Logger) *Installer {
	configDir := path.Join(DataDir, "config")
	executor := NewExecutor(logger)
	return &Installer{
		newVersionFile:     path.Join(AppDir, "version"),
		currentVersionFile: path.Join(DataDir, "version"),
		configDir:          configDir,
		platformClient:     platform.New(),
		installFile:        path.Join(CommonDir, "installed"),
		executor:           executor,
		logger:             logger,
	}
}

func (i *Installer) Install() error {
	return i.UpdateConfigs()
}

func (i *Installer) Configure() error {
	if i.IsInstalled() {
		return i.Upgrade()
	}
	return i.Initialize()
}

func (i *Installer) IsInstalled() bool {
	_, err := os.Stat(i.installFile)
	return err == nil
}

func (i *Installer) Initialize() error {
	if err := i.StorageChange(); err != nil {
		return err
	}
	if err := os.WriteFile(i.installFile, []byte("installed"), 0644); err != nil {
		return err
	}
	return i.UpdateVersion()
}

func (i *Installer) Upgrade() error {
	if err := i.StorageChange(); err != nil {
		return err
	}
	return i.UpdateVersion()
}

func (i *Installer) PreRefresh() error {
	return nil
}

func (i *Installer) PostRefresh() error {
	if err := i.UpdateConfigs(); err != nil {
		return err
	}
	if err := i.ClearVersion(); err != nil {
		return err
	}
	return i.FixPermissions()
}

func (i *Installer) AccessChange() error {
	return i.UpdateConfigs()
}

func (i *Installer) StorageChange() error {
	storageDir, err := i.platformClient.InitStorage(App, App)
	if err != nil {
		return err
	}
	if err := i.createMissingDirs(
		path.Join(DataDir, "storage"),
		path.Join(DataDir, "servers"),
	); err != nil {
		return err
	}
	return linux.Chown(storageDir, App)
}

func (i *Installer) ClearVersion() error {
	return os.RemoveAll(i.currentVersionFile)
}

func (i *Installer) UpdateVersion() error {
	return cp.Copy(i.newVersionFile, i.currentVersionFile)
}

func (i *Installer) UpdateConfigs() error {
	if err := linux.CreateUser(App); err != nil {
		return err
	}
	if err := i.StorageChange(); err != nil {
		return err
	}
	if err := createMissingDir(path.Join(DataDir, "nginx")); err != nil {
		return err
	}

	authUrl, err := i.platformClient.GetAppUrl("auth")
	if err != nil {
		return err
	}

	if err := i.registerOIDC(); err != nil {
		i.logger.Warn("oidc register failed (continuing with forward-auth)", zap.Error(err))
	}

	variables := Variables{
		AuthUrl:         authUrl,
		AuthLocalSocket: i.platformClient.GetAuthLocalSocket(),
	}

	if err := config.Generate(
		path.Join(AppDir, "config"),
		path.Join(DataDir, "config"),
		variables,
	); err != nil {
		return err
	}

	return i.FixPermissions()
}

func (i *Installer) registerOIDC() error {
	password, err := i.platformClient.RegisterOIDCClient(App, "/auth/callback", true, "client_secret_basic")
	if err != nil {
		return err
	}
	return os.WriteFile(path.Join(DataDir, "oidc.secret"), []byte(password), 0640)
}

func (i *Installer) FixPermissions() error {
	if err := linux.Chown(DataDir, App); err != nil {
		return err
	}
	return linux.Chown(CommonDir, App)
}

func (i *Installer) BackupPreStop() error  { return i.PreRefresh() }
func (i *Installer) RestorePreStart() error { return i.PostRefresh() }
func (i *Installer) RestorePostStart() error { return i.Configure() }

func (i *Installer) createMissingDirs(dirs ...string) error {
	for _, dir := range dirs {
		if err := createMissingDir(dir); err != nil {
			i.logger.Error("cannot create dir", zap.String("dir", dir), zap.Error(err))
			return err
		}
	}
	return nil
}

func createMissingDir(dir string) error {
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return os.Mkdir(dir, 0755)
	}
	return nil
}
