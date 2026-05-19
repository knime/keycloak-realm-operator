#!groovy

// Jenkinsfile for building an application artifact and a docker image.

library "knime-pipeline@$DEFAULT_LIBRARY_VERSION"

def IMAGE = 'knime/keycloak-realm-operator'
def HARBOR_REG = 'registry.hubdev.knime.com'

properties([
    buildDiscarder(logRotator(numToKeepStr: '5')),
    disableConcurrentBuilds(),
    parameters([booleanParam(defaultValue: false, description: 'Whether this is a release build', name: 'RELEASE_BUILD')])
])


timeout(time: 15, unit: 'MINUTES') {
  node('docker') {
    try {

      def version

      stage('Checkout Sources') {
        env.lastStage = env.STAGE_NAME
        cleanWs()
        checkout scm
        knimetools.reportJIRAIssues()
      }

      stage('Build and Test') {
        parallel(

          'Docker': {

            stage('Build Docker Image') {
              env.lastStage = env.STAGE_NAME
              def timestamp = new Date().format("yyyy-MM-dd'T'HH:mm:ss'Z'", TimeZone.getTimeZone("UTC"))
              dir(knimetools.jenkinsfileDir()) {
                dockerTools.build(IMAGE, "--build-arg CACHE_DATE=${timestamp}")
              }
            }

            if (BRANCH_NAME == "master" || BRANCH_NAME.startsWith("releases/") || BRANCH_NAME.startsWith("feature/") || BRANCH_NAME.startsWith("fix/")) {
              if (currentBuild.result != 'UNSTABLE' || params.FORCE_DEPLOYMENT) {
                dir(knimetools.jenkinsfileDir()) {
                  version = readFile('VERSION').trim() + changelistSuffix()
                }
                def dockerFriendlyVersion = version.replace('+', '-')
                stage('Push Image') {
                  env.lastStage = env.STAGE_NAME
                  dockerTools.push(IMAGE, "latest,${dockerFriendlyVersion}")

                  if (env.BRANCH_NAME ==~ /^(?:master|main|releases\/.+|fix\/.+)$/) {
                      dockerTools.pushToHarborHub(IMAGE, dockerFriendlyVersion)
                  }
                }
              } else {
                echo "============================================\n" +
                    "| Build unstable, not deploying artifacts. |\n" +
                    "============================================"
              }
            }

            stage('Remove Local Images') {
              env.lastStage = env.STAGE_NAME
              sh "docker rmi -f ${IMAGE}"
              sh '''
                list=$(docker images -q -f "dangling=true" -f "label=autodelete=true")
                if [ -n "$list" ]; then
                     docker rmi -f $list
                fi
              '''
            }
          }

        )
      }

    } catch (ex) {
      currentBuild.result = 'FAILURE'
      throw ex
    } finally {
      notifications.notifyBuild(buildStatus: currentBuild.result);
    }
  }
}

def changelistSuffix() {
    def commitHash = sh (
        returnStdout: true,
        script: 'git rev-parse --short HEAD'
    ).trim()

    def commitTs = sh (
        returnStdout: true,
        script: 'date -d @$(git show -s --format=format:%ct) +%Y%m%d-%H%M%S'
    ).trim();

    if (params.RELEASE_BUILD == true) {
        return "-stable-" + String.format("%04d", env.BUILD_NUMBER as Integer) + "-${commitHash}"
    } else if (BRANCH_NAME == "master") {
        return "-beta-${commitTs}-" + String.format("%06d", env.BUILD_NUMBER as Integer) + "-${commitHash}"
    } else if (BRANCH_NAME.startsWith("releases/")) {
        return "-rc-" + String.format("%06d", env.BUILD_NUMBER as Integer) +  "-${commitHash}"
    } else {
        return "-alpha-" + BRANCH_NAME.replace("/", "-") + "-" + String.format("%06d", env.BUILD_NUMBER as Integer) + "-${commitHash}"
    }
}