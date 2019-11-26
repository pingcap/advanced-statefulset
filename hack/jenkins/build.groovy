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
  - name: docker
    image: docker:19.03.1
    command:
    - sleep
    args:
    - 99d
    env:
      - name: DOCKER_HOST
        value: tcp://localhost:2375
  - name: docker-daemon
    image: docker:19.03.1-dind
    securityContext:
      privileged: true
    env:
      - name: DOCKER_TLS_CERTDIR
        value: ""
''') {
    node(POD_LABEL) {
        container('docker') {
            stage('Checkout repository') {
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
            }

			stage("Basic checks") {
				sh "./hack/run-in-docker.sh make verify build test test-integration"
			}

			stage("Test") {
				sh "./hack/run-in-docker.sh make test"
			}

			stage("Integration") {
				sh "./hack/run-in-docker.sh make test-integration"
			}

			stage("E2E") {
				sh "make e2e-v1.16.1"
			}
        }
    }
}

// vim: noet
