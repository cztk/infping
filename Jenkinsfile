@Library('jenkings') _

def getNodeHostname() {
  script {
    sh 'hostname -f'
  }
}

def getGitTag() {
  dir ('src/github.com/amgxv/infping') {
    println('trying to get git tag')
    GIT_TAG = sh(script: 'git describe --abbrev=0 --tags', returnStdout: true).trim()
    println('detected tag : '+GIT_TAG)
  }  
}

def buildDockerImage(image,arch,tag) {
  dir ('src/github.com/amgxv/infping') {
    script {
      img = docker.build("${image}:${arch}-${tag}", "--pull --build-arg ARCH=${arch} -f docker/Dockerfile .")
    }
  }    
}

def pushDockerImage(arch,tag) {
  def githubImage = "docker.pkg.github.com/amgxv/infping/infping-$arch:$tag"
  script {
    docker.withRegistry('https://registry.hub.docker.com', 'amgxv_dockerhub') { 
      img.push("${arch}-${tag}")  
    }
    docker.withRegistry('https://docker.pkg.github.com', 'amgxv-github-token') {
      sh "docker image tag ${img.id} ${githubImage}"
      docker.image(githubImage).push()  
    }    
  }
}

def goBuild(arch, gopath) {
  def envVars = ['GOPATH=' + gopath, "GOARCH=${arch}", 'GOCACHE=' + gopath + '/.go-cache']
  if(arch == 'arm32v7') {
    envVars.add 'GOARM=7'
  }

  withEnv(envVars) {
    sh 'go get -d -v ./...'
    sh 'go build -o infping-' + arch + ' -v ./...'
  }
}

def checkoutAndBuild(arch) {
  def gopath = pwd()
  dir ('src/github.com/amgxv/infping') {
    GIT_COMMIT_DESCRIPTION = sh(script: 'git log -n1 --pretty=format:%B | cat', returnStdout: true).trim()
    echo "current description : ${GIT_COMMIT_DESCRIPTION}"
    goBuild(arch, gopath)
    archiveArtifacts artifacts: "infping-${arch}", fingerprint: true
  }  
}

def uploadArtifacts() {
  println('artifact with tag : '+ GIT_TAG)
  unarchive mapping: ['*': '.']  
  githubRelease(
    'amgxv-github-token',
    'amgxv/infping',
    "${GIT_TAG}",
    "infping - Release ${GIT_TAG}",
    'Parse fping output, store result in influxdb - '+ GIT_COMMIT_DESCRIPTION,
    [
      ['infping-amd64', 'application/octet-stream'],
      ['infping-arm', 'application/octet-stream'],
      ['infping-arm64', 'application/octet-stream']
    ]
  )
}


pipeline {
  agent { label '!docker-qemu' }
  options { checkoutToSubdirectory('src/github.com/amgxv/infping')}
  environment {
    img = ''
    image = 'amgxv/infping'
    chatId = credentials('amgxv-telegram-chatid')
    GIT_COMMIT_DESCRIPTION = ''
    GIT_TAG = ''
  }

  stages {

    stage('Preparation') {
      steps {
        telegramSend 'infping is being built [here](' + env.BUILD_URL + ')... ', chatId
      }
    }

    stage ('Build Docker Images') {

        parallel {
          stage ('docker-amd64') {
            agent {
              label 'docker-qemu'
            }

            environment {
              arch = 'amd64'
              tag = 'latest'
              img = null
            }

            stages {

              stage ('Current Node') {
                steps {
                  getNodeHostname()
                }
              }

              stage ('Build') {
                steps {
                  buildDockerImage(image,arch,tag)
                }
              }

              stage ('Push') {
                steps {
                  pushDockerImage(arch,tag)
                }
              }
            }
          }

          stage ('docker-arm64') {
            agent {
              label 'docker-qemu'
            }

            environment {
              arch = 'arm64v8'
              tag = 'latest'
              img = null
            }

            stages {

              stage ('Current Node') {
                steps {
                  getNodeHostname()
                }
              }

              stage ('Build') {
                steps {
                  buildDockerImage(image,arch,tag)
                }
              }

              stage ('Push') {
                steps {
                  pushDockerImage(arch,tag)
                }
              }
            }
          }

          stage ('docker-arm32') {
            agent {
              label 'docker-qemu'
            }

            environment {
              arch = 'arm32v7'
              tag = 'latest'
              img = null
            }

            stages {

              stage ('Current Node') {
                steps {
                  getNodeHostname()
                }
              }

              stage ('Build') {
                steps {
                  buildDockerImage(image,arch,tag)
                }
              }

              stage ('Push') {
                steps {
                  pushDockerImage(arch,tag)
                }
              }
            }
          }
        }
      }  

    stage('Update manifest (Only Dockerhub)') {

      agent {
        label 'docker-qemu'
      }

      steps {
        script {
          docker.withRegistry('https://index.docker.io/v1/', 'amgxv_dockerhub') {
            getNodeHostname()  
            sh 'docker manifest create amgxv/infping:latest amgxv/infping:amd64-latest amgxv/infping:arm32v7-latest amgxv/infping:arm64v8-latest'
            sh 'docker manifest push -p amgxv/infping:latest'
          }        
        }
      }
    }

    stage ('Build Go Binaries') {
      parallel {

        stage ('amd64') {
          agent {
            docker {
              image 'golang:buster'
            }
          }

          environment {
            arch = 'amd64'
            tag = 'latest'
          }

          steps {
            checkoutAndBuild(arch)
          }
        }

        stage ('arm') {
          agent {
            docker {
              image 'golang:buster'
            }
          }

          environment {
            arch = 'arm'
            tag = 'latest'
          }

          steps {
            checkoutAndBuild(arch)
          }
        }

        stage ('arm64') {
          agent {
            docker {
              image 'golang:buster'
            }
          }

          environment {
            arch = 'arm64'
            tag = 'latest'
          }

          steps {
            checkoutAndBuild(arch)
          }
        }                            
      }
    }

  stage ('Get Git Tag'){
    agent { label 'majorcadevs'}
    steps {
      getGitTag()
    }
  }

  stage('Upload Release to Github') {
      agent { label 'majorcadevs'}
      when { expression { "${GIT_TAG}" != null } }
      steps {
          uploadArtifacts()
      }
    }    
  }    
    
  post {
    success {
      cleanWs()
      telegramSend 'infping has been built successfully. ', chatId
    }
    failure {
      cleanWs()
      telegramSend 'infping could not have been built.\n\nSee [build log](' + env.BUILD_URL + ')... ', chatId
    }
  }
}