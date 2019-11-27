//
// Bootstrap Jenkins Pipeline Script
//
node {
    println("ghprbPullId: " + ghprbPullId)
    println("ghprbPullTitle: " + ghprbPullTitle)
    println("ghprbPullLink: " + ghprbPullLink)
    println("ghprbPullDescription: "+ ghprbPullDescription)
   
    def CREDENTIALS_ID = "github-sre-bot-ssh"
    def GIT_URL = "git@github.com:pingcap/advanced-statefulset.git"
    def GIT_REF = "${ghprbActualCommit}"

    checkout changelog: false,
        poll: false,
        scm: [
        $class: 'GitSCM',
        branches: [[name: "${GIT_REF}"]],
        doGenerateSubmoduleConfigurations: false,
        extensions: [],
        submoduleCfg: [],
        userRemoteConfigs: [[
            credentialsId: "${CREDENTIALS_ID}",
            refspec: '+refs/heads/*:refs/remotes/origin/* +refs/pull/*:refs/remotes/origin/pr/*',
            url: "${GIT_URL}",
            ]]
        ]
    
    def jenkins = load "hack/jenkins/build.groovy"
    jenkins.call(GIT_URL, GIT_REF)
}

// vim: et
