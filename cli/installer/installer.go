package installer

import (
	"fmt"
	cp "github.com/otiai10/copy"
	"github.com/syncloud/golib/config"
	"github.com/syncloud/golib/linux"
	"github.com/syncloud/golib/platform"
	"go.uber.org/zap"
	"os"
	"path"
)

const (
	App       = "games"
	AppDir    = "/snap/games/current"
	DataDir   = "/var/snap/games/current"
	CommonDir = "/var/snap/games/common"
)

type Variables struct {
	AuthUrl string
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
	// Game install dirs go under storage (/data/games/servers/) so they
	// survive snap refresh rollback and aren't included in platform backups.
	if err := i.createMissingDirs(
		path.Join(storageDir, "servers"),
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
		return fmt.Errorf("oidc register: %w", err)
	}

	if err := i.trustSyncloudCA(); err != nil {
		i.logger.Warn("syncloud CA install failed (backend has in-process fallback)", zap.Error(err))
	}

	variables := Variables{
		AuthUrl: authUrl,
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

// trustSyncloudCA copies the platform's self-signed CA into the system
// trust store and runs update-ca-certificates so that anything in this
// snap doing TLS (Go's stdlib http.Client, curl in install scripts, …)
// validates auth.<domain> cleanly. Same pattern platform/test uses to
// seed the CA into a test image.
func (i *Installer) trustSyncloudCA() error {
	src := "/var/snap/platform/current/syncloud.ca.crt"
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("syncloud CA missing: %w", err)
	}
	dst := "/usr/local/share/ca-certificates/syncloud.crt"
	if err := cp.Copy(src, dst); err != nil {
		return fmt.Errorf("install CA: %w", err)
	}
	return i.executor.Run("/usr/sbin/update-ca-certificates")
}

func (i *Installer) registerOIDC() error {
	password, err := i.platformClient.RegisterOIDCClient(App, "/auth/callback", true, "client_secret_basic")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path.Join(DataDir, "oidc.secret"), []byte(password), 0640); err != nil {
		return err
	}
	authUrl, err := i.platformClient.GetAppUrl("auth")
	if err != nil {
		return err
	}
	appUrl, err := i.platformClient.GetAppUrl(App)
	if err != nil {
		return err
	}
	cfg := fmt.Sprintf(`{"authUrl":%q,"clientId":%q,"clientSecret":%q,"redirectUrl":%q}`,
		authUrl, App, password, appUrl+"/auth/callback")
	return os.WriteFile(path.Join(DataDir, "oidc.json"), []byte(cfg), 0640)
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
