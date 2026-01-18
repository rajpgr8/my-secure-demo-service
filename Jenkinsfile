pipeline {
    agent any

    environment {
        // --- CONFIGURATION ---
        // TODO: Update with your actual registry URL
        REGISTRY_URL = 'my-registry.example.com' 
        
        // Repo Name Updated
        IMAGE_NAME   = 'my-secure-demo-service'
        
        // Use Jenkins Build Number for immutable tags
        IMAGE_TAG    = "${BUILD_NUMBER}" 
        FULL_IMAGE   = "${REGISTRY_URL}/${IMAGE_NAME}:${IMAGE_TAG}"
        
        // --- CREDENTIALS IDs (Must match Jenkins Credentials) ---
        // 1. Docker Registry (Username/Password)
        DOCKER_CREDS_ID = 'docker-registry-creds'
        
        // 2. Cosign Private Key (Secret File)
        COSIGN_KEY_ID = 'cosign-private-key'
        
        // 3. Cosign Password (Secret Text)
        COSIGN_PASSWORD_ID = 'cosign-password'
    }

    stages {
        stage('Checkout SCM') {
            steps {
                checkout scm
            }
        }

        stage('Build (Distroless)') {
            steps {
                script {
                    echo "--- 🔨 Building Docker Image ---"
                    // Best Practice: --no-cache ensures we pick up the latest OS patches in the base image
                    sh "docker build --no-cache -t ${FULL_IMAGE} ."
                }
            }
        }

        stage('Generate SBOM (Syft)') {
            steps {
                script {
                    echo "--- 🔍 Generating Software Bill of Materials ---"
                    // We run Syft as a container, mounting the docker socket.
                    // This allows Syft to inspect the image we just built without installing Syft on the agent.
                    sh """
                    docker run --rm \
                        -v /var/run/docker.sock:/var/run/docker.sock \
                        -v \$(pwd):/out \
                        anchore/syft \
                        ${FULL_IMAGE} -o cyclonedx-json --file /out/sbom.json
                    """
                    
                    // Archive SBOM so you can download it from the Jenkins UI for audit
                    archiveArtifacts artifacts: 'sbom.json', fingerprint: true
                }
            }
        }

        stage('Security Scan (Trivy)') {
            steps {
                script {
                    echo "--- 🚨 Scanning for Vulnerabilities ---"
                    
                    // 1. Generate Human-Readable Report (Does not fail build)
                    sh """
                    docker run --rm \
                        -v /var/run/docker.sock:/var/run/docker.sock \
                        aquasec/trivy image \
                        --format table \
                        --severity CRITICAL,HIGH \
                        --ignore-unfixed \
                        ${FULL_IMAGE}
                    """

                    // 2. The Gate (Fails Build on Critical)
                    // We wrap this to fail the pipeline if Trivy returns exit code 1
                    sh """
                    docker run --rm \
                        -v /var/run/docker.sock:/var/run/docker.sock \
                        aquasec/trivy image \
                        --severity CRITICAL \
                        --exit-code 1 \
                        --ignore-unfixed \
                        --quiet \
                        ${FULL_IMAGE}
                    """
                }
            }
        }

        stage('Push to Registry') {
            steps {
                script {
                    echo "--- ⬆️ Pushing Image ---"
                    // BEST PRACTICE: Fetch Credentials only in this block
                    withCredentials([usernamePassword(credentialsId: env.DOCKER_CREDS_ID, usernameVariable: 'REGISTRY_USER', passwordVariable: 'REGISTRY_PASS')]) {
                        // Use stdin to pass password securely (avoids process list exposure)
                        sh "echo \$REGISTRY_PASS | docker login ${REGISTRY_URL} -u \$REGISTRY_USER --password-stdin"
                        sh "docker push ${FULL_IMAGE}"
                    }
                }
            }
        }

        stage('Sign & Attest (Cosign)') {
            steps {
                script {
                    echo "--- 🔏 Signing Artifacts ---"
                    // BEST PRACTICE: Fetch Key File and Password strictly for this step
                    withCredentials([
                        file(credentialsId: env.COSIGN_KEY_ID, variable: 'COSIGN_KEY_PATH'), 
                        string(credentialsId: env.COSIGN_PASSWORD_ID, variable: 'COSIGN_PASSWORD')
                    ]) {
                        // 1. Sign the Image Digest
                        sh """
                        docker run --rm \
                            -v \$(pwd):/workspace \
                            -v \$COSIGN_KEY_PATH:/key.key \
                            -e COSIGN_PASSWORD=\$COSIGN_PASSWORD \
                            bitnami/cosign \
                            sign --key /key.key -y ${FULL_IMAGE}
                        """

                        // 2. Attest the SBOM (Upload SBOM to registry attached to image)
                        sh """
                        docker run --rm \
                            -v \$(pwd):/workspace \
                            -v \$COSIGN_KEY_PATH:/key.key \
                            -e COSIGN_PASSWORD=\$COSIGN_PASSWORD \
                            bitnami/cosign \
                            attest --key /key.key --type cyclonedx --predicate /workspace/sbom.json -y ${FULL_IMAGE}
                        """
                    }
                }
            }
        }
    }

    post {
        always {
            script {
                echo "--- 🧹 Cleaning Workspace ---"
                // Clean up the heavy image to save Jenkins agent disk space
                sh "docker rmi ${FULL_IMAGE} || true"
                
                // Always logout to prevent credential reuse by other jobs
                sh "docker logout ${REGISTRY_URL} || true"
            }
        }
        success {
            echo "✅ Pipeline Succeeded: ${FULL_IMAGE} is secured and published."
        }
        failure {
            echo "❌ Pipeline Failed. Check logs for details."
        }
    }
}
