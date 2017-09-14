# duo-bot [![Download](https://api.bintray.com/packages/palantir/releases/duo-bot/images/download.svg)](https://bintray.com/palantir/releases/duo-bot/_latestVersion)

A simple app to keep temporary state on arbitrary keys, and if that key has had someone accept a [DUO MFA](https://duo.com/product) action against it.

Duo-bot is packaged as a go binary in bintray (link above) as well as a dockerhub image (details on running server below) `palantirtechnologies/duo-bot`

## State

As of today, this app does not store state anywhere except in memory.  This was done purely to keep the app simple during development.  This has two major consequences:

* Restarts of duo-bot cause total state loss.  If you run this in a container scheduler like [nomad](https://www.nomadproject.io/intro/index.html), keep in mind that any reschedule or failover of the application will cause any stored keys to be lost and must be regenerated.  Given the highly-ephemeral nature of the data duo-bot stores, this probably won't be an issue.  If a `check` request fails to find a key which was accepted recently because of a restart of the app, the prompt simply needs to be reissued.
* Duo-bot can only have one instance running.  Given that the state is not external to the running application, there's no way for multiple instances of the application to keep state in sync between them, so running multiple copies of this app behind a load-balancer means repeated requests could return inconsistent results.

## Usage

The following examples assume you're interested in tracking whether the key `MYKEY` has had someone DUO against it.  Replace this with the HEAD or your git hash or whatever you'd like to track.

### Create a new key or reset status of existing key

* Issue an async DUO push to a user
  * `curl -X POST 'http://ADDR/v1/push/MYKEY?user=USERNAME&async=1'`
* Enter a DUO passcode
  * `curl -X POST 'http://ADDR/v1/passcode/MYKEY?user=USERNAME&passcode=123456'`
* Issue a blocking DUO push to a user
  * `curl -X POST 'http://ADDR/v1/push/MYKEY?user=USERNAME'`
* Add extra metadata to the DUO push
  * `curl -X POST -H 'Content-Type: application/json' -d '{ "duoPushInfo": "key1=val1&key2=val2&key3=otherthing" }' 'http://ADDR/v1/push/MYKEY?user=USERNAME'`

### Check the status of a key

* To just get a `0` or `1` exitcode
  * `curl --fail http://ADDR/v1/check/MYKEY?user=USERNAME`
* Omit the `--fail` if you'd desire more output at the expense of losing the correct exitcode.
* If you don't care _who_ MFA'd your key, just that it was MFA'd, you can omit the `user` flag.
  * `curl --fail http://ADDR/v1/check/MYKEY`

## Running the server

* The server expects a config file name to be passed-in with the `-c` parameter (see `./duo-bot --help`).  This config file should look like this.

```yml
duo:
  host: "api-???.duosecurity.com"
  ikey: "???"
  skey: "???"
```

* Note too that the server doesn't support SSL for its http listener.  The expectation here is that you run an ELB, nginx proxy or something else in front of duo-bot which terminates client SSL connections.
* To run the server via the docker image, write your config file as per above into its own directory, and name it `duo-bot.yml`.  Mount that directory to `/secrets/` in the docker image.

```bash
cat > /tmp/duo-bot-config/duo-bot.yml << EOF
duo:
  host: "api-???.duosecurity.com"
  ikey: "???"
  skey: "???"
EOF
docker run --rm -v /tmp/duo-bot-config:/secrets/ -p <LOCAL PORT>:8080 palantirtechnologies/duo-bot:(<RELEASE>|latest)
```

## Applications

* A git pre-receive hook.
  * See [mfa-protect.sh](examples/mfa-protect.sh) for a script that works with github enterprise.  Simply configure a global pre-receive hook that invokes this script (probably default it to off).
    * The `.duo.whitelist` is a file that can be added to the repo to exclude changes to certain files from requiring a DUO authentication to alter.  Each line of this file should be a regex of files that do _not_ require MFA to alter on the default branch.
  * Any repo which enables this pre-receive hook will issue a DUO push to anyone trying to alter the default branch of the repo, either via a direct `git push` or via a pull request.

## Contributing

For general guidelines on contributing the Palantir products, see [this page](https://github.com/palantir/gradle-baseline/blob/develop/docs/best-practices/contributing/readme.md)
