FROM scratch

ARG ARCH
EXPOSE 9550

COPY dist/trojan-go-exporter_linux_${ARCH} /usr/bin/trojan-go-exporter
ENTRYPOINT [ "/usr/bin/trojan-go-exporter" ]
