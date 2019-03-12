FROM golang:1.12

# Install system dependecies
#
RUN apt-get update \
    && apt-get -y install --no-install-recommends \
        make

ARG DIR=$(pwd)
ADD . /firectl
WORKDIR /firectl
CMD TRIMPATH="$DIR" make release && cp firectl /release
