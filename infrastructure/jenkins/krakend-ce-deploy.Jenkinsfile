@Library('jenkins-shared-libraries') _

deployPipeline(
    projectName: "krakend-ce",
    language: "go",
    builderDockerImage: "golang:1.25-bookworm",
    builderDockerImagePrivateRegistry: false,
    packageOnly: true,
    slackChannel: "#builds-platform",
    disableStableDeploy: true
)