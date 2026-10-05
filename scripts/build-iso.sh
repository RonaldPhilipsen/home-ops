#!/bin/bash

docker run --rm -t \
  --security-opt label=disable \
  --security-opt seccomp=unconfined \
  -v "$PWD/talos/_out/iso:/out" \
  ghcr.io/siderolabs/imager:v1.14.1@sha256:95e76e773cf8258af2f6e8ac8fae04ce27350051f33fcbda767561c6de7a5884 \
  iso \
  --extra-kernel-arg talos.dashboard.disabled=1 \
  --system-extension-image ghcr.io/siderolabs/intel-ucode@sha256:87a21b3cf3db7c81db3ec2f5c85e38b65f50214c45929309c9addae4897465f0 \
  --system-extension-image ghcr.io/siderolabs/iscsi-tools@sha256:97794a8b6064e24d8053b6d424662ccd844052ee57e1b3efc795889ec21bcb0b \
  --system-extension-image ghcr.io/siderolabs/realtek-firmware@sha256:c4bb59a33701a14224698fbfe8bba01bd5c0064f5b2ef45938e58b65b595cc69 \
  --system-extension-image ghcr.io/siderolabs/util-linux-tools@sha256:304c463024a9e3f427262d847e0a71b5d7548d29fcabd5259a79f405ac15842d \
  --system-extension-image ghcr.io/rothgar/seatd@sha256:3a0a8003e92917dd896709e156e1a51a2060656c7d805264c33b94603b6d47e0 \
