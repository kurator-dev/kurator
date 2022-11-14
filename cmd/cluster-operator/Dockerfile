ARG BASE_VERSION
ARG BASE_IMAGE
FROM ${BASE_IMAGE}:${BASE_VERSION}

COPY cluster-operator /usr/local/bin/
RUN chmod 750 /usr/local/bin/cluster-operator
ENTRYPOINT ["/usr/local/bin/cluster-operator"]
