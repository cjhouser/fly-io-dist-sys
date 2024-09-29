FROM golang@sha256:ac67716dd016429be8d4c2c53a248d7bcdf06d34127d3dc451bda6aa5a87bc06 AS build
WORKDIR /build
COPY go.mod go.sum ./
COPY *.go ./
RUN go build -a \
    -tags purego \
    -ldflags "-extldflags '-static' -s -w" \
    -o node .

FROM fedora@sha256:d0207dbb078ee261852590b9a8f1ab1f8320547be79a2f39af9f3d23db33735e
RUN dnf -y install \
    java-17-openjdk-17.0.12.0.7-2.fc40 \
    graphviz-go-9.0.0-11.fc40 \
    gnuplot-5.4.9-3.fc40 \
    && dnf clean all
RUN curl --remote-header-name --location --output maelstrom.tar.bz2 \
    https://github.com/jepsen-io/maelstrom/releases/download/v0.2.3/maelstrom.tar.bz2 \
    && tar --extract --verbose --bzip2 --file maelstrom.tar.bz2 \
    && rm maelstrom.tar.bz2
WORKDIR /maelstrom
COPY --from=build /build/node /maelstrom/node
RUN mkdir -p /root/go/bin/
RUN ln -s /maelstrom/node /root/go/bin/maelstrom-echo
RUN ln -s /maelstrom/node /root/go/bin/maelstrom-unique-ids
RUN ln -s /maelstrom/node /root/go/bin/maelstrom-broadcast 
CMD [ "./maelstrom" ]