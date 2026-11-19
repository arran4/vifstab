# GoReleaser dockers_v2 supplies already-built static binaries.
# scratch has no external base image, package manager, shell or editor.
FROM scratch
ARG TARGETOS
ARG TARGETARCH
COPY --chmod=0755 ${TARGETOS}/${TARGETARCH}/vifstab /usr/local/bin/vifstab
COPY LICENSE THIRD_PARTY_NOTICES.md /usr/share/doc/vifstab/
USER 65532:65532
ENTRYPOINT ["/usr/local/bin/vifstab"]
