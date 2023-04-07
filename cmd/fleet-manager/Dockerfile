ARG BASE_VERSION
ARG BASE_IMAGE
FROM ${BASE_IMAGE}:${BASE_VERSION}

# Use uid of nonroot user (65532) because kubernetes expects numeric user when applying pod security policies
USER 65532

WORKDIR /
COPY fleet-manager .
ENTRYPOINT ["/fleet-manager"]
