//
// Jenkins pipeline script for release job.
//
// It accepts two parameters:
//
// - ghprbActualCommit (string): git commit to build
// - ghprbPullId (string): pull request ID to build
//
// These two parameters are populated by sre-bot.
//

def BUILD_URL = "git@github.com:pingcap/advanced-statefulset.git"
def BUILD_BRANCH = "${ghprbActualCommit}"

podTemplate(yaml: '''
apiVersion: v1
kind: Pod
spec:
  containers:
  - name: main
    image: gcr.io/k8s-testimages/kubekins-e2e:v20191108-9467d02-master
    command:
    - runner.sh
    - sleep
    - 99d
    # we need privileged mode in order to do docker in docker
    securityContext:
      privileged: true
    env:
    - name: DOCKER_IN_DOCKER_ENABLED
      value: "true"
    resources:
      requests:
        memory: "4000Mi"
        cpu: 2000m
    # kind needs /lib/modules and cgroups from the host
    volumeMounts:
    - mountPath: /lib/modules
      name: modules
      readOnly: true
    - mountPath: /sys/fs/cgroup
      name: cgroup
  volumes:
  - name: modules
    hostPath:
      path: /lib/modules
      type: Directory
  - name: cgroup
    hostPath:
      path: /sys/fs/cgroup
      type: Directory
''') {
    node(POD_LABEL) {
        container('main') {
            stage("Debug Info") {
                println "debug command: kubectl -n jenkins-ci exec -ti ${NODE_NAME} bash"
            }
            stage('Build') {
                dir("/home/jenkins/agent/workspace/go/src/github.com/pingcap/advanced-statefulset") {
                    checkout changelog: false,
                        poll: false,
                        scm: [
                            $class: 'GitSCM',
                            branches: [[name: "${BUILD_BRANCH}"]],
                            doGenerateSubmoduleConfigurations: false,
                            extensions: [],
                            submoduleCfg: [],
                            userRemoteConfigs: [[
                                credentialsId: 'github-sre-bot-ssh',
                                refspec: '+refs/heads/*:refs/remotes/origin/* +refs/pull/*:refs/remotes/origin/pr/*',
                                url: "${BUILD_URL}",
                            ]]
                        ]
                    sh """
                    echo "====== shell env ======"
                    echo "pwd: \$(pwd)"
                    env
                    echo "====== go env ======"
                    go env
                    echo "====== docker version ======"
                    docker version
                    export GOPATH=/home/jenkins/agent/workspace/go
                    make verify build test test-integration
                    make e2e
                    """
                }
            }
        }
    }
}

// vim: et
