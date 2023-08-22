<h1> GOCT </h1>

Yet another Certificate Transparency monitor and checker.

And yet another project written just to learn one more programming language, but might be helpful for somebody :) 

<h2> Features </h2>

1) Custom checks support with custom configs 
2) Configs which covers different types of CT logs
3) Runs without database
4) Could be deployed as cloud func like AWS Lambda

<h1> Ideas for checks </h1>

* Regex matches for CN's
* Invalid (corrupted) certificates
* Absence of log entries in several national CT logs

<h1> How to run: </h1>

* Run checks by default and exit (could be used as cloud function by cron)
``` bash
TELEGRAM_APITOKEN=... DEBUG=false VERBOSE=false ./goct --config config.yaml
```

* Run as a daemon (all checks will be performed every rescan value in seconds)
``` bash
TELEGRAM_APITOKEN=... VERBOSE=false ./goct daemon --rescan 3600 --config config.yaml
```
Daemon mode also supports simple http healthchecks on localhost:8081/ping

* Run as cli

```bash
./goct cli --config config.yaml --logUri https://ctlog2024.mail.ru/nca2024/ --lookupDepth 175
```

Config example:
```yaml
---
version: 1
verbose: false
numWorkers: 1
batchSize: 100
daemon: false
checks:
  - name: match_by_regexp
    regex:
      - ".*bank.*"
      # re2 regexp to filter out domains with *.ru zone
      # - $.+(.{0,4}$)|(\.[^r].{0,2}$)|(\.r[^u].{0,2}$)|(\.ru.{1,4})$
    logs:
      - "https://ct.googleapis.com/logs/us1/argon2024/"
    lookupDepth: 24 #hours
  - name: invalid_cert
    logs:
      - "https://ct-agate.yandex.net/2024"
      # - "https://ct.googleapis.com/logs/us1/argon2024/"
    lookupDepth: 24
  - name: recently_issued_cert
    logs:
      # - "https://ct-agate.yandex.net/2024"
      - "https://ctlog2024.mail.ru/nca2024/"
      # - "https://ct.googleapis.com/logs/us1/argon2024/"
    lookupDepth: 1
    lookupDelta: 100

store:
  - type: "sqlite"
    tableName: "certs"
    uri: "file://tmp/1.db"
    flush: true

notifications:
  - type: telegram
    recipients:
    # telegram chat ids
      - 1337
```

<h1> TODO's </h1>

* Generic secrets provisioning (not only through env's)
* More notifications clients (not only telegram)
* More DB's clients (not only sqlite)
* Your issue :)
