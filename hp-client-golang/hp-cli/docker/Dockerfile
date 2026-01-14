FROM alpine:latest
ARG TARGETARCH
ARG TARGETVARIANT
WORKDIR /hp-lite
COPY --chmod=777 target/hp-lite-${TARGETARCH}${TARGETVARIANT} /hp-lite/hp-lite
CMD ["./hp-lite"]