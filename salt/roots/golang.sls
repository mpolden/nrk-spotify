curl:
  pkg:
    - installed

curl -L http://golang.org/dl/go1.3.linux-amd64.tar.gz | tar -zxC /usr/local:
  cmd.run:
    - unless: test -d /usr/local/go

/etc/profile.d/golang.sh:
  file.managed:
    - source: salt://files/golang.sh
    - user: root
    - group: root
    - mode: 0644