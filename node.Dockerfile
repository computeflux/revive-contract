# wetee-node
FROM ubuntu:24.04

## ubuntu update
RUN apt-get update && apt install -y ca-certificates

## copy bin from builder
COPY  /target/node /usr/local/bin
COPY  /target/eth-rpc /usr/local/bin


EXPOSE 30333 9933 9944 9615
ENTRYPOINT ["/usr/local/bin/node","--dev","--rpc-external","--rpc-methods=unsafe","--unsafe-rpc-external","--rpc-cors=all","--base-path","/chain-data"]