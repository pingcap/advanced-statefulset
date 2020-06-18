//
// Jenkins pipeline script for release job.
//
// It accepts two parameters:
//
// - BUILD_REF (string): git ref to build
// - IMAGE_TAG (string): image tag to build
//

def REPO_GIT_URL = "git@github.com:pingcap/advanced-statefulset.git"

try {
    node('delivery') {
        container("delivery") {
            def WORKSPACE = pwd()
            deleteDir()
            stage('Checkout repository'){
                checkout changelog: false, poll: false, scm: [$class: 'GitSCM', branches: [[name: "${BUILD_REF}"]], doGenerateSubmoduleConfigurations: false, extensions: [], submoduleCfg: [], userRemoteConfigs: [[credentialsId: 'github-sre-bot-ssh', refspec: '+refs/pull/*:refs/remotes/origin/pr/*', url: "${REPO_GIT_URL}"]]]
            }

            stage('Build and push docker image'){
                withDockerServer([uri: "tcp://localhost:2375"]) {
                    def image = docker.build("pingcap/advanced-statefulset:${IMAGE_TAG}")
                    image.push()
                    docker.withRegistry("https://registry.cn-beijing.aliyuncs.com", "ACR_TIDB_ACCOUNT") {
                        sh "docker tag pingcap/advanced-statefulset:${IMAGE_TAG} registry.cn-beijing.aliyuncs.com/tidb/advanced-statefulset:${IMAGE_TAG}"
                        sh "docker push registry.cn-beijing.aliyuncs.com/tidb/advanced-statefulset:${IMAGE_TAG}"
                    }
                }
            }
        }
    }
    currentBuild.result = "SUCCESS"
} catch (err) {
    currentBuild.result = 'FAILURE'
}

stage('Summary') {
    echo("######## Summary info ########")
    def DURATION = ((System.currentTimeMillis() - currentBuild.startTimeInMillis) / 1000 / 60).setScale(2, BigDecimal.ROUND_HALF_UP)
    def slackmsg = "[${env.JOB_NAME.replaceAll('%2F','/')}-${env.BUILD_NUMBER}] `${currentBuild.result}`" + "\n" +
    "Elapsed Time: `${DURATION}` Mins" + "\n" +
    "Repo: pingcap/advanced-statefulset" + "\n" +
    "Git ref: `${BUILD_REF}`" + "\n" +
    "Display URL:" + "${env.RUN_DISPLAY_URL}" + "\n"
    def color = "good"

    if (currentBuild.result != "SUCCESS") {
        color = "danger"
    } else {
        slackmsg = "${slackmsg}" + "\n" + "advanced-statefulset Docker Image: `pingcap/advanced-statefulset:${IMAGE_TAG}`" + "\n"
    }

    echo(color)
    echo(slackmsg)
    slackSend channel: '#cloud_jenkins', color: color, teamDomain: 'pingcap', tokenCredentialId: 'slack-pingcap-token', message: "${slackmsg}"
}

// vim: et
