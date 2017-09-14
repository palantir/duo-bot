#!/bin/bash

set -u

# Protect the default branch by requiring that any changes to it require the user
# making them to DUO.  This applies to direct pushes, and all types of merge from the github UI.

# Workflow:
#   Perform action that alters default branch
#   Action fails and prints message below about sending a DUO push
#   Accept DUO push
#   Perform same action as before, and this time it should succeed

URL='https://ADDR/v1'
default_branch="$(git symbolic-ref HEAD)"

key=""

cacert=/tmp/cacert.pem

# Plug in here the CA cert that has signed the certificate for the ELB or proxy
# etc that terminates ssl for this script hitting duo-bot.
cat > $cacert << 'EOF'
-----BEGIN CERTIFICATE-----
...
-----END CERTIFICATE-----
EOF

zero_commit="0000000000000000000000000000000000000000"

# For each updated ref...
while read oldrev newrev refname; do
    if [[ "${default_branch}" == "${refname}" ]]; then
        echo "Attempting to write to default branch, MFA enforcement triggered"
        if [[ "${newrev}" == "${zero_commit}" ]]; then
            # Deleting the default branch - don't imagine this will happen very often
            key="${oldrev}"
        else
            key="$(git rev-parse ${newrev}^{tree})"
        fi
        break
    fi
done

if [[ "${key:-x}" != "x" ]]; then
    whitelist_bypass=false
    git show ${oldrev}:.duo.whitelist >/tmp/${key}_whitelist 2>/dev/null
    if [ "$?" == "0" ]; then
        # There's a whitelist... check it
        git diff --name-only $oldrev $newrev > /tmp/${key}_diffs

        # For each whitelist pattern, remove matching files
        # If at the end, there are no more matching files, then
        # all files in diff must match at least one regex in the whitelist
        while read regex; do
            egrep -v "$regex" /tmp/${key}_diffs >/tmp/${key}_diffs2 2>/dev/null
            mv /tmp/${key}_diffs2 /tmp/${key}_diffs
        done < /tmp/${key}_whitelist

        remaining=$(wc -l < /tmp/${key}_diffs)
        if [ "$remaining" == "0" ]; then
            whitelist_bypass=true
        fi
    fi

    if [ "$whitelist_bypass" == "true" ]; then
        echo "MFA whitelist policy allows these changes"
        exit 0
    fi

    curl \
        -s \
        --cacert $cacert \
        --fail \
        "${URL}/check/${key}?user=${GITHUB_USER_LOGIN}"

    ec="$?"
    echo ""

    if [[ "${ec}" == "0" ]]; then
        echo "Valid MFA acceptance found from ${GITHUB_USER_LOGIN}"
        exit 0
    else
        echo "No Valid MFA found for key ${key}"
        echo "Sending DUO push to user ${GITHUB_USER_LOGIN}"
        echo "Accept DUO push and try again"

        extraMeta="Application=Github Enterprise&repository=${GITHUB_REPO_NAME}"
        if [[ "${GITHUB_VIA:-x}" != "x" ]]; then
            extraMeta="${extraMeta}&Method=${GITHUB_VIA}"
        fi

        curl \
            -s \
            -o /dev/null \
            --cacert $cacert \
            -X POST \
            -H 'Content-Type: application/json' \
            -d "{ \"duoPushInfo\": \"${extraMeta}\" }" \
            "${URL}/push/${key}?user=${GITHUB_USER_LOGIN}&async=1"

        exit 1
    fi
fi

exit 0
