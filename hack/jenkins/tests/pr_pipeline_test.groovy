// Run from repository root: groovy hack/jenkins/tests/pr_pipeline_test.groovy
List checkouts = []
List calls = []
Binding bootstrapBinding = new Binding([
    ghprbActualCommit: 'a' * 40,
    ghprbPullId: '1',
    ghprbPullTitle: 'test',
    ghprbPullLink: 'https://github.com/pingcap/advanced-statefulset/pull/1',
    ghprbPullDescription: 'test',
    node: { Closure c -> c() },
    dir: { String p, Closure c -> assert p == 'trusted-pipeline'; c() },
    deleteDir: {},
    checkout: { Map args -> checkouts << args.scm },
    load: { String path ->
        assert path == 'hack/jenkins/build.groovy'
        [call: { String url, String ref -> calls << [url, ref] }]
    },
    error: { String msg -> throw new IllegalArgumentException(msg) }
])
new GroovyShell(bootstrapBinding).evaluate(new File('hack/jenkins/bootstrap.groovy'))
assert checkouts.size() == 1
assert checkouts[0].branches == [[name: 'refs/remotes/origin/master']]
assert checkouts[0].userRemoteConfigs == [[url: 'https://github.com/pingcap/advanced-statefulset.git', refspec: '+refs/heads/master:refs/remotes/origin/master']]
assert calls == [['https://github.com/pingcap/advanced-statefulset.git', 'a' * 40]]
bootstrapBinding.setVariable('ghprbActualCommit', 'master; injected')
try {
    new GroovyShell(bootstrapBinding).evaluate(new File('hack/jenkins/bootstrap.groovy'))
    assert false: 'invalid SHA accepted'
} catch (IllegalArgumentException expected) {}
assert checkouts.size() == 1

List commands = []
List pods = []
checkouts.clear()
Binding buildBinding = new Binding([
    POD_LABEL: 'test-agent',
    podTemplate: { Map args, Closure c -> pods << args.yaml; c() },
    node: { String label, Closure c -> c() },
    container: { String name, Closure c -> assert name == 'main'; c() },
    dir: { String path, Closure c -> assert path == 'source'; c() },
    checkout: { Map args -> checkouts << args.scm },
    sh: { String code -> commands << code },
    timeout: { int value, Closure c -> c() },
    stage: { String name, Closure c -> c() },
    parallel: { Map branches -> branches.values().findAll { it instanceof Closure }.each { it() } },
    echo: { String msg -> assert msg.contains('GitHub Actions') },
    error: { String msg -> throw new IllegalArgumentException(msg) }
])
def pipeline = new GroovyShell(buildBinding).evaluate(new File('hack/jenkins/build.groovy'))
pipeline.call('https://github.com/pingcap/advanced-statefulset.git', 'b' * 40)
assert commands == ['make verify', 'make build test', 'make test-integration']
assert checkouts.size() == 3 && checkouts.every { it.branches == [[name: 'b' * 40]] }
assert checkouts.every { !it.userRemoteConfigs[0].containsKey('credentialsId') }
assert pods.size() == 3
assert pods.every { !it.contains('hostPath:') && !it.contains('privileged: true') && !it.contains('DOCKER_IN_DOCKER') }
assert pods.every { it.contains('automountServiceAccountToken: false') && it.contains('runAsNonRoot: true') && it.contains('drop: [ALL]') }
try {
    pipeline.call('https://attacker.invalid/project.git', 'b' * 40)
    assert false: 'foreign repository accepted'
} catch (IllegalArgumentException expected) {}
assert commands.size() == 3
println 'PR pipeline boundary regressions passed'
