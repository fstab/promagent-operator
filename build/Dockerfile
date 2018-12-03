FROM alpine:3.8

RUN apk upgrade --update --no-cache

ADD https://github.com/fstab/promagent/releases/download/v1.0-SNAPSHOT/promagent.jar https://github.com/fstab/promagent/releases/download/v1.0-SNAPSHOT/promagent-loader-1.0-SNAPSHOT.jar /
RUN chmod 644 /promagent.jar /promagent-loader-1.0-SNAPSHOT.jar

USER nobody

ADD build/_output/bin/promagent-operator /usr/local/bin/promagent-operator
