local name = 'games';
local go = '1.25';
local nginx = '1.24.0';
local node = '20';
local platform = '26.04.7';
local python = '3.12-slim-bookworm';
local deployer = 'https://github.com/syncloud/store/releases/download/4/syncloud-release';
local distros = ['bookworm'];
local distro_default = 'bookworm';
local arch = 'amd64';

[{
  kind: 'pipeline',
  type: 'docker',
  name: arch,
  platform: {
    os: 'linux',
    arch: arch,
  },
  steps: [
    {
      name: 'version',
      image: 'alpine:3.17.0',
      commands: [
        'echo $DRONE_BUILD_NUMBER > version',
      ],
    },
    {
      name: 'nginx',
      image: 'nginx:' + nginx,
      commands: [
        './nginx/build.sh',
      ],
    },
    {
      name: 'nginx test',
      image: 'syncloud/platform-' + distro_default + '-' + arch + ':' + platform,
      commands: [
        './nginx/test.sh',
      ],
    },
    {
      name: 'web',
      image: 'node:' + node,
      commands: [
        './web/build.sh',
      ],
    },
    {
      name: 'catalog',
      image: 'golang:' + go,
      commands: [
        './catalog/build.sh',
      ],
    },
    {
      name: 'jre',
      image: 'debian:bookworm-slim',
      commands: [
        './jre/build.sh',
      ],
    },
    {
      name: 'backend',
      image: 'golang:' + go,
      commands: [
        './backend/build.sh',
      ],
    },
    {
      name: 'steamcmd',
      image: 'debian:bookworm-slim',
      commands: [
        './steamcmd/build.sh',
      ],
    },
    {
      name: 'cli',
      image: 'golang:' + go,
      commands: [
        'cd cli',
        'mkdir -p ../build/snap/meta/hooks',
        'CGO_ENABLED=0 go build -buildvcs=false -o ../build/snap/meta/hooks/install ./cmd/install',
        'CGO_ENABLED=0 go build -buildvcs=false -o ../build/snap/meta/hooks/configure ./cmd/configure',
        'CGO_ENABLED=0 go build -buildvcs=false -o ../build/snap/meta/hooks/pre-refresh ./cmd/pre-refresh',
        'CGO_ENABLED=0 go build -buildvcs=false -o ../build/snap/meta/hooks/post-refresh ./cmd/post-refresh',
        'CGO_ENABLED=0 go build -buildvcs=false -o ../build/snap/bin/cli ./cmd/cli',
      ],
    },
    {
      name: 'package',
      image: 'debian:bookworm-slim',
      commands: [
        'VERSION=$(cat version)',
        './package.sh ' + name + ' $VERSION',
      ],
    },
  ] + [
    {
      name: 'test ' + distro,
      image: 'python:' + python,
      commands: [
        'DOMAIN="' + distro + '.com"',
        'APP_DOMAIN="' + name + '.' + distro + '.com"',
        'getent hosts $APP_DOMAIN | sed "s/$APP_DOMAIN/auth.$DOMAIN/g" | tee -a /etc/hosts',
        'cat /etc/hosts',
        'APP_ARCHIVE_PATH=$(realpath $(cat package.name))',
        'cd test',
        './deps.sh',
        'py.test -x -s test.py --distro=' + distro + ' --app-archive-path=$APP_ARCHIVE_PATH --app=' + name + ' --arch=' + arch,
      ],
    }
    for distro in distros
  ] + [
    {
      name: 'test-ui-' + projectName,
      image: 'mcr.microsoft.com/playwright:v1.59.1-jammy',
      commands: [
        'DOMAIN="' + distro_default + '.com"',
        'APP_DOMAIN="' + name + '.' + distro_default + '.com"',
        'getent hosts $APP_DOMAIN | sed "s/$APP_DOMAIN/auth.$DOMAIN/g" | tee -a /etc/hosts',
        'cat /etc/hosts',
        'PLAYWRIGHT_DOMAIN=' + distro_default + '.com ./ci/ui.sh ' + projectName,
      ],
    }
    for projectName in ['desktop', 'mobile']
  ] + [
    {
      name: 'upload',
      image: 'debian:bookworm-slim',
      environment: {
        AWS_ACCESS_KEY_ID: { from_secret: 'AWS_ACCESS_KEY_ID' },
        AWS_SECRET_ACCESS_KEY: { from_secret: 'AWS_SECRET_ACCESS_KEY' },
        SYNCLOUD_TOKEN: { from_secret: 'SYNCLOUD_TOKEN' },
      },
      commands: [
        'PACKAGE=$(cat package.name)',
        'apt update && apt install -y wget',
        'wget ' + deployer + '-' + arch + ' -O release --progress=dot:giga',
        'chmod +x release',
        './release publish -f $PACKAGE -b $DRONE_BRANCH',
      ],
      when: {
        branch: ['stable', 'master'],
        event: ['push'],
      },
    },
    {
      name: 'promote',
      image: 'debian:bookworm-slim',
      environment: {
        AWS_ACCESS_KEY_ID: { from_secret: 'AWS_ACCESS_KEY_ID' },
        AWS_SECRET_ACCESS_KEY: { from_secret: 'AWS_SECRET_ACCESS_KEY' },
        SYNCLOUD_TOKEN: { from_secret: 'SYNCLOUD_TOKEN' },
      },
      commands: [
        'apt update && apt install -y wget',
        'wget ' + deployer + '-' + arch + ' -O release --progress=dot:giga',
        'chmod +x release',
        './release promote -n ' + name + ' -a $(dpkg --print-architecture)',
      ],
      when: {
        branch: ['stable'],
        event: ['push'],
      },
    },
    {
      name: 'artifact',
      image: 'appleboy/drone-scp:1.6.4',
      settings: {
        host: { from_secret: 'artifact_host' },
        username: 'artifact',
        key: { from_secret: 'artifact_key' },
        timeout: '2m',
        command_timeout: '2m',
        target: '/home/artifact/repo/' + name + '/${DRONE_BUILD_NUMBER}-' + arch,
        source: 'artifact/*',
        strip_components: 1,
      },
      when: {
        status: ['failure', 'success'],
        event: ['push'],
      },
    },
  ],
  trigger: {
    event: ['push', 'pull_request'],
  },
  services: [
    {
      name: name + '.' + distro + '.com',
      image: 'syncloud/platform-' + distro + '-' + arch + ':' + platform,
      privileged: true,
      volumes: [
        { name: 'dbus', path: '/var/run/dbus' },
        { name: 'dev', path: '/dev' },
      ],
    }
    for distro in distros
  ],
  volumes: [
    { name: 'dbus', host: { path: '/var/run/dbus' } },
    { name: 'dev', host: { path: '/dev' } },
  ],
}]
