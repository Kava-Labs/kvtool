FROM ubuntu:18.04

RUN apt-get update && apt-get install -y ca-certificates wget moreutils jq curl

ARG BVER=0.6.3
ARG CLIVER=0.6.3

RUN wget -O /usr/local/bin/bnbchaind https://github.com/binance-chain/node-binary/raw/master/fullnode/prod/${BVER}/linux/bnbchaind
RUN wget -O /usr/local/bin/bnbcli https://github.com/binance-chain/node-binary/raw/master/cli/prod/${CLIVER}/linux/bnbcli
RUN chmod u+x /usr/local/bin/bnbchaind /usr/local/bin/bnbcli
