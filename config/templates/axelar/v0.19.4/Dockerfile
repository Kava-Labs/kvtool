FROM ubuntu:22.04

RUN apt-get update && apt-get install -y ca-certificates wget moreutils jq curl nginx sudo

RUN wget -O /usr/local/bin/axelard https://github.com/axelarnetwork/axelar-core/releases/download/v0.19.4/axelard-linux-arm64-v0.19.4
RUN wget -O /usr/local/bin/tofnd https://github.com/axelarnetwork/tofnd/releases/download/v0.10.1/tofnd-linux-arm64-v0.10.1
RUN chmod u+x /usr/local/bin/axelard /usr/local/bin/tofnd

RUN mkdir -p /root/.axelar/data
RUN echo '{"height":"0","round":0,"step":0}' > /root/.axelar/data/priv_validator_state.json

COPY initstate/entrypoint.sh /entrypoint.sh
ENTRYPOINT [ "/entrypoint.sh" ]
