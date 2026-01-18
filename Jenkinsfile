pipeline {
    agent any

    environment {
        // --- CONFIGURATION ---
        REGISTRY_URL = 'my-registry'
        IMAGE_NAME   = 'my-secure-service'
        FULL_IMAGE   = "\${REGISTRY_URL}/\${IMAGE_NAME}:\${BUILD_NUMBER}"
        
        // --- CREDENTIALS IDs (Create these in Jenkins) ---
        // 1. Docker Registry Creds (Username/Password)
        DOCKER_CREDS = 'docker-registry-creds'
        
        // 2. Cosign Private Key (Secret File)
        COSIGN_KEY_ID = 'cosign-private-key'
        
        // 3. Cosign Password (Secret Text)
        COSIGN_PASSWORD_ID = 'cosign-password'
    }

    stages {
        stage('Build 🏗️') {
            steps {
                script {
                    echo "--- Building Distroless Image ---"
                    sh "docker build -t \${FULL_IMAGE} ."
                }
            }
        }

        stage('SBOM 📜') {
            steps {
                script {
                    echo "--- Generating SBOM (Syft) ---"
                    // Run Syft as a container so we don't need it installed on the agent
                    sh """
                    docker run --rm \\
                        -v /var/run/docker.sock:/var/run/docker.sock \\
                        -v \$(pwd):/out \\
                        anchore/syft \\
                        \${FULL_IMAGE} -o cyclonedx-json --file /out/sbom.json
                    """
                    archiveArtifacts artifacts: 'sbom.json'
                }
            }
        }

        stage('Scan 🛡️') {
            steps {
                script {
                    echo "--- Scanning for Vulnerabilities (Trivy) ---"
                    // Fail build if CRITICAL vulns found
                    // We scan the SBOM generated in the previous step
                    sh """
                    docker run --rm \\
                        -v \$(pwd):/in \\
                        aquasec/trivy image \\
                        --input /in/sbom.json \\
                        --format table \\
                        --severity CRITICAL \\
                        --exit-code 1 \\
                        --ignore-unfixed
                    """
                }
            }
        }

        stage('Login & Push ☁️') {
            steps {
                withCredentials([usernamePassword(credentialsId: env.DOCKER_CREDS, usernameVariable: 'USER', passwordVariable: 'PASS')]) {
                    sh "echo \$PASS | docker login \$REGISTRY_URL -u \$USER --password-stdin"
                    sh "docker push \${FULL_IMAGE}"
                }
            }
        }

        stage('Sign ✍️') {
            steps {
                withCredentials([
                    file(credentialsId: env.COSIGN_KEY_ID, variable: 'COSIGN_KEY_PATH'), 
                    string(credentialsId: env.COSIGN_PASSWORD_ID, variable: 'COSIGN_PASSWORD')
                ]) {
                    script {
                        echo "--- Signing Image & Attesting SBOM (Cosign) ---"
                        // Run Cosign container
                        // mapping the key file into the container
                        sh """
                        docker run --rm \\
                            -v \$(pwd):/workspace \\
                            -v \$COSIGN_KEY_PATH:/key.key \\
                            -e COSIGN_PASSWORD=\$COSIGN_PASSWORD \\
                            bitnami/cosign \\
                            sign --key /key.key -y \${FULL_IMAGE}
                        """

                        echo "--- Attesting SBOM ---"
                        sh """
                        docker run --rm \\
                            -v \$(pwd):/workspace \\
                            -v \$COSIGN_KEY_PATH:/key.key \\
                            -e COSIGN_PASSWORD=\$COSIGN_PASSWORD \\
                            bitnami/cosign \\
                            attest --key /key.key --type cyclonedx --predicate /workspace/sbom.json -y \${FULL_IMAGE}
                        """
                    }
                }
            }
        }
    }

    post {
        always {
            // Clean up docker images to save space
            sh "docker rmi \${FULL_IMAGE} || true"
        }
        failure {
            echo "❌ Pipeline Failed! Check the logs."
        }
    }
}
