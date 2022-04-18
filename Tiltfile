if 'ENABLE_NGROK_EXTENSION' in os.environ and os.environ['ENABLE_NGROK_EXTENSION'] == '1':
  v1alpha1.extension_repo(
    name = 'default',
    url = 'https://github.com/tilt-dev/tilt-extensions'
  )
  v1alpha1.extension(name = 'ngrok', repo_name = 'default', repo_path = 'ngrok')

load('ext://min_k8s_version', 'min_k8s_version')
min_k8s_version('1.18.0')

load('ext://namespace', 'namespace_create')
namespace_create('brigade-acr-gateway')
k8s_resource(
  new_name = 'namespace',
  objects = ['brigade-acr-gateway:namespace'],
  labels = ['brigade-acr-gateway']
)

docker_build(
  'brigadecore/brigade-acr-gateway', '.',
  only = [
    'internal/',
    'config.go',
    'go.mod',
    'go.sum',
    'main.go'
  ],
  ignore = ['**/*_test.go']
)
k8s_resource(
  workload = 'brigade-acr-gateway',
  new_name = 'gateway',
  port_forwards = '31700:8080',
  labels = ['brigade-acr-gateway']
)
k8s_resource(
  workload = 'gateway',
  objects = [
    'brigade-acr-gateway:secret',
    'brigade-acr-gateway-config:secret'
  ]
)

k8s_yaml(
  helm(
    './charts/brigade-acr-gateway',
    name = 'brigade-acr-gateway',
    namespace = 'brigade-acr-gateway',
    set = [
      'brigade.apiToken=' + os.environ['BRIGADE_API_TOKEN'],
      'tls.enabled=false',
      'tokens.dev-token=insecure-dev-token'
    ]
  )
)
