# 🛡️ Secure Distroless Service Template

A production-ready "Golden Repository" template for building, securing, and deploying Go microservices on **Distroless** images. 

This project implements a **Defense-in-Depth** strategy, ensuring that security is not an "afterthought" but an integral part of the build pipeline ("Shift Left").

---

## 🏗️ Architecture & Decisions

### 1. The Base Image: Distroless
**Decision:** We use \`gcr.io/distroless/static-debian12\`.
* **Why:** It contains *no shell*, *no package manager*, and *no system utilities*. 
* **Benefit:** Reduces the attack surface by ~95% compared to standard Ubuntu/Alpine images. Vulnerability scanners (Trivy/Grype) typically report 0-2 CVEs instead of 50+.

### 2. Multi-Stage Build
**Decision:** We separate the *Build Environment* from the *Runtime Environment*.
* **Stage 1 (Builder):** \`golang:1.21-alpine\`. Contains compilers, git, and build tools.
* **Stage 2 (Runtime):** \`distroless/static\`. Contains **only** the compiled binary.
* **Security:** \`CGO_ENABLED=0\` ensures the binary is statically linked and requires no external C libraries (like \`glibc\` or \`musl\`).

### 3. "No-Exec" Probes
**Decision:** We use \`httpGet\` probes instead of \`exec\`.
* **Problem:** Distroless has no \`/bin/sh\`, so standard \`exec: ["cat", "/tmp/health"]\` probes fail.
* **Solution:**
    * **Startup Probe:** Protects slow-starting apps from being killed early.
    * **Readiness Probe:** Checks DB/Cache connections before accepting traffic.
    * **Liveness Probe:** Simple "ping" to detect deadlocks.

### 4. Supply Chain Security (The "Seamless Loop")
**Decision:** We trust nothing implicitly. Every build undergoes a rigorous loop:
1.  **Catalog:** \`Syft\` generates an SBOM (Software Bill of Materials).
2.  **Scan:** \`Trivy\` scans the *SBOM* for vulnerabilities (more accurate for Distroless).
3.  **Sign:** \`Cosign\` cryptographically signs the image and attaches the SBOM.
4.  **Verify:** (Optional) Cluster admission controllers reject unsigned images.

---

## 📂 Repository Structure

```text
.
├── app/                     # Source Code
│   ├── main.go              # Application entrypoint (handles SIGTERM)
│   └── go.mod               # Dependencies
├── k8s/                     # Kubernetes Manifests
│   ├── deployment.yaml      # Includes Startup/Readiness/Liveness probes
│   └── network-policy.yaml  # "Default Deny" firewall rules
├── security/                # Runtime Security
│   └── falco-rules.yaml     # Falco rules for Distroless anomalies
├── Dockerfile               # Multi-stage build definition
└── Makefile                 # Automation for the "Seamless Loop"

```

## 🚀 Quick Start

### Prerequisites
* Docker
* Go 1.21+
* [Syft](https://github.com/anchore/syft) (SBOM Generator)
* [Trivy](https://github.com/aquasecurity/trivy) (Scanner)
* [Cosign](https://github.com/sigstore/cosign) (Signing Tool)

### 1. Initialize Keys
Run this **once** to generate your signing keys (\`cosign.key\` and \`cosign.pub\`).
```bash
make keygen
```

### 2. The "Seamless Loop" Build
Run the full pipeline. This will **Build -> Catalog -> Scan -> Sign**.
```bash
make all
```

> **Note:** The build will **fail** if \`Trivy\` detects CRITICAL vulnerabilities. This is a deliberate "Security Gate."

---

## 🛡️ Operational Security (Runtime)

### 1. Debugging (Ephemeral Containers)
Since you cannot SSH into the container, use **Kubernetes Ephemeral Containers** to debug.

```bash
# Attach a debug shell (alpine) to the running pod
kubectl debug -it <pod-name> \\
  --image=nicolaka/netshoot \\
  --target=app \\
  -- sh

# Access files via the shared process namespace
cd /proc/1/root/
```

### 2. Network Firewall (Zero Trust)
The \`k8s/network-policy.yaml\` enforces a **Default Deny** stance.
* **Ingress:** Blocked by default. Only allowed from specific sources (e.g., Ingress Controller).
* **Egress:** Blocked by default. We explicitly allow ONLY:
    * DNS (UDP 53) - *Required for service discovery.*
    * Postgres (TCP 5432) - *Or your specific backend.*

### 3. Falco Detection (The "Highlander" Rule)
Since Distroless containers should only ever run **one** process (the app), any other spawned process is a confirmed anomaly.

* **Rule:** \`Distroless Anomaly - Unexpected Process\`
* **Trigger:** If an attacker manages to run \`ls\`, \`cat\`, or execute a dropped binary.
* **Action:** Triggers a \`CRITICAL\` alert to your SIEM/Slack.

---

## 📝 Developer Cheat Sheet

| Task | Command | Why? |
| :--- | :--- | :--- |
| **Build Locally** | \`make build\` | Creates the docker image. |
| **Generate SBOM** | \`make sbom\` | Creates \`sbom.json\` listing all packages. |
| **Check Security** | \`make scan\` | Fails if known CVEs exist in the image. |
| **Sign Image** | \`make sign\` | Signs the image for production trust. |
| **View Logs** | \`kubectl logs -f <pod>\` | Standard logging (stdout/stderr). |
| **Debug Pod** | \`kubectl debug ...\` | "Sidecar" debugging (see above). |
