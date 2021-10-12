FROM golang:1.15.6 AS builder

RUN mkdir /tmp/go

# Copy plugins source code
COPY transformers /tmp/go/transformers
COPY lib /tmp/go/lib
COPY go.mod /tmp/go/

# Compile kustomize plugins
WORKDIR /tmp/go/
RUN CGO_ENABLED=0 GOOS=linux go build -o transformers/DataReplaceInline/DataReplaceInline transformers/DataReplaceInline/DataReplaceInline.go

FROM argoproj/argocd:v1.8.1

# Switch to root for the ability to perform install
USER root

# Create plugins directories
RUN mkdir -p /home/argocd/.config/kustomize/plugin/transformers.kustomize.com/v1/datareplaceinline

# Copy plugins from builder image
COPY --from=builder /tmp/go/transformers/DataReplaceInline/DataReplaceInline /home/argocd/.config/kustomize/plugin/transformers.kustomize.com/v1/datareplaceinline/

RUN chmod +x /home/argocd/.config/kustomize/plugin/transformers.kustomize.com/v1/datareplaceinline/DataReplaceInline && \
    chown -R argocd. /home/argocd/.config

## Install tools needed for your repo-server to retrieve & decrypt secrets, render manifests
## (e.g. curl, awscli, gpg, sops)
RUN apt-get update && \
    apt-get install -y curl vim && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/* && \
    curl -s "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"  | bash && \
    mv -f kustomize /usr/local/bin/ && \
    chmod +x /usr/local/bin/kustomize

# Switch back to non-root user
USER argocd
