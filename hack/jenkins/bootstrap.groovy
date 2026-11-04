// Configure this bootstrap from a trusted branch in the Jenkins job, never a PR.
node {
    def gitUrl = 'https://github.com/pingcap/advanced-statefulset.git'
    def gitRef = "${ghprbActualCommit}"
    if (!(gitRef ==~ /[0-9a-fA-F]{40}/)) {
        error('PR commit must be a full Git SHA')
    }
    def pipeline
    dir('trusted-pipeline') {
        deleteDir()
        checkout changelog: false, poll: false, scm: [
            $class: 'GitSCM',
            branches: [[name: 'refs/remotes/origin/master']],
            userRemoteConfigs: [[url: gitUrl, refspec: '+refs/heads/master:refs/remotes/origin/master']]
        ]
        pipeline = load 'hack/jenkins/build.groovy'
    }
    pipeline.call(gitUrl, gitRef)
}
