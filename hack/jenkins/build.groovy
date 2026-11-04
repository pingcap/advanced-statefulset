// Loaded only from the trusted master checkout; PR contents are build inputs.
import groovy.transform.Field

@Field
String podYAML = '''
apiVersion: v1
kind: Pod
spec:
  automountServiceAccountToken: false
  securityContext:
    runAsNonRoot: true
    runAsUser: 1000
    runAsGroup: 1000
    fsGroup: 1000
  containers:
  - name: main
    image: golang:1.23.2
    workingDir: /home/jenkins/agent
    command: [sleep, 99d]
    securityContext:
      allowPrivilegeEscalation: false
      capabilities:
        drop: [ALL]
      seccompProfile:
        type: RuntimeDefault
    env:
    - name: GOCACHE
      value: /tmp/go-build
    - name: GOMODCACHE
      value: /tmp/go-mod
    resources:
      requests:
        memory: 2000Mi
        cpu: 1000m
      limits:
        memory: 4000Mi
'''

def build(gitUrl, gitRef, shellCode) {
    podTemplate(yaml: podYAML) {
        node(POD_LABEL) {
            container('main') {
                dir('source') {
                    checkout changelog: false, poll: false, scm: [
                        $class: 'GitSCM',
                        branches: [[name: gitRef]],
                        userRemoteConfigs: [[
                            url: gitUrl,
                            refspec: '+refs/heads/*:refs/remotes/origin/* +refs/pull/*:refs/remotes/origin/pr/*'
                        ]]
                    ]
                    sh shellCode
                }
            }
        }
    }
}

def call(gitUrl, gitRef) {
    if (gitUrl != 'https://github.com/pingcap/advanced-statefulset.git' || !(gitRef ==~ /[0-9a-fA-F]{40}/)) {
        error('Only a full commit SHA in the public project repository is allowed')
    }
    timeout(60) {
        stage('Verify') {
            build(gitUrl, gitRef, 'make verify')
        }
        parallel([
            'Build and Test': { build(gitUrl, gitRef, 'make build test') },
            'Integration': { build(gitUrl, gitRef, 'make test-integration') },
            failFast: false
        ])
        echo('kind E2E runs in GitHub Actions on disposable hosted VMs; it is not run in privileged Jenkins pods.')
    }
}

return this
