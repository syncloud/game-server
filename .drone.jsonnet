local name = 'games';
local go = '1.25';
local nginx = '1.24.0';
local node = '20';
local platform = '26.04.10';
local python = '3.12-slim-bookworm';
local deployer = 'https://github.com/syncloud/store/releases/download/4/syncloud-release';
// Binary smoke tests run on every distro we ship for, full integration
// (pytest, playwright) only on the default distro.
local distros = ['bookworm', 'buster'];
local distro_default = 'bookworm';
local arch = 'amd64';

local platform_image(distro, arch) =
  'syncloud/platform-' + distro + '-' + arch + ':' + platform;

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
  ] + [
    {
      name: 'nginx test ' + distro,
      image: platform_image(distro, arch),
      commands: ['./nginx/test.sh'],
    }
    for distro in distros
  ] + [
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
  ] + [
    {
      name: 'jre test ' + distro,
      image: platform_image(distro, arch),
      commands: ['./jre/test.sh'],
    }
    for distro in distros
  ] + [
    {
      name: 'steamcmd',
      image: 'debian:bookworm-slim',
      commands: [
        './steamcmd/build.sh',
      ],
    },
  ] + [
    {
      name: 'steamcmd test ' + distro,
      image: platform_image(distro, arch),
      commands: ['./steamcmd/test.sh'],
    }
    for distro in distros
  ] + [
    {
      name: 'backend',
      image: 'golang:' + go,
      commands: [
        './backend/build.sh',
      ],
    },
    {
      name: 'cli',
      image: 'golang:' + go,
      commands: [
        './cli/build.sh',
      ],
    },
  ] + [
    {
      name: 'cli test ' + distro,
      image: platform_image(distro, arch),
      commands: ['./cli/test.sh'],
    }
    for distro in distros
  ] + [
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
      name: 'test ' + distro_default,
      image: 'python:' + python,
      commands: [
        'DOMAIN="' + distro_default + '.com"',
        'APP_DOMAIN="' + name + '.' + distro_default + '.com"',
        'getent hosts $APP_DOMAIN | sed "s/$APP_DOMAIN/auth.$DOMAIN/g" | tee -a /etc/hosts',
        'cat /etc/hosts',
        'APP_ARCHIVE_PATH=$(realpath $(cat package.name))',
        'cd test',
        './deps.sh',
        'py.test -x -s test.py --distro=' + distro_default + ' --app-archive-path=$APP_ARCHIVE_PATH --app=' + name + ' --arch=' + arch,
      ],
    },
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
      name: name + '.' + distro_default + '.com',
      image: platform_image(distro_default, arch),
      privileged: true,
      volumes: [
        { name: 'dbus', path: '/var/run/dbus' },
        { name: 'dev', path: '/dev' },
      ],
    },
  ],
  volumes: [
    { name: 'dbus', host: { path: '/var/run/dbus' } },
    { name: 'dev', host: { path: '/dev' } },
  ],
}]
