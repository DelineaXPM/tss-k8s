FROM tss-injector:latest

ARG cert_file
ARG key_file
ARG roles_file

COPY ${cert_file} ./tss.pem
COPY ${key_file} ./tss.key
COPY ${roles_file} ./roles.json

ENTRYPOINT ["tss-injector-svc", "-cert", "tss.pem", "-key", "tss.key", "-roles", "roles.json" ]
