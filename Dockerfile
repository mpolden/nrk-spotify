FROM ubuntu:14.04

# Time zone
ENV DEBIAN_FRONTEND noninteractive
RUN echo "Europe/Oslo" > /etc/timezone
RUN dpkg-reconfigure tzdata

# Install ca-certificates
RUN apt-get -y update
RUN apt-get -y install ca-certificates

# Add app
RUN mkdir /app
ADD bin/nrk-spotify /app/nrk-spotify
RUN chmod 0755 /app/nrk-spotify
ENTRYPOINT ["/app/nrk-spotify"]
