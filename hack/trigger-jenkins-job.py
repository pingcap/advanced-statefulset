#!/usr/bin/env python3
# encoding: utf-8

"""
Script to trigger jenkins job.
"""

import argparse
import urllib.request
import urllib.parse
import urllib.error
import sys

JOB_URL = "https://internal.pingcap.net/idc-jenkins/job/release_advanced_statefulset/buildWithParameters"


def trigger(url, params={}):
    query = urllib.parse.urlencode(params)
    url = "{}?{}".format(JOB_URL, query)
    sys.stderr.write("GET {}\n".format(url))
    req = urllib.request.Request(url)
    with urllib.request.urlopen(req) as f:
        sys.stderr.write("Response.Code: {}\n".format(f.getcode()))
        if f.getcode() in [200, 201]:
            sys.stderr.write("Success\n")
            sys.exit(0)
        else:
            sys.stderr.write("Failed\n")
            sys.exit(1)
    return


# ref examples:
# - refs/heads/master
# - refs/tags/v1.0.0
def tag_from_ref(ref):
    seps=ref.split("/")
    if len(seps) <= 0:
        raise Exception("unexpected ref: {}".ref)
    ref=seps[len(seps)-1]
    if ref == "master":
        return "latest"
    else:
        return ref


if __name__ == '__main__':
    parser=argparse.ArgumentParser()
    parser.add_argument('--token', required=True)
    parser.add_argument('--build-ref', required=True)
    parser.add_argument('--image-tag')
    options=parser.parse_args()
    params={}
    params["token"]=options.token
    params["BUILD_REF"]=options.build_ref
    if options.image_tag:
        params["IMAGE_TAG"]=options.image_tag
    else:
        params["IMAGE_TAG"]=tag_from_ref(options.build_ref)
    trigger(JOB_URL, params)
