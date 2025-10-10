FROM alpine:latest as certs
RUN apk --update add ca-certificates

FROM scratch
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY device-admin /
COPY web/templates/ /web/templates
ENV TEMPLATES_PATH=/web/templates
ENTRYPOINT ["/device-admin"]
