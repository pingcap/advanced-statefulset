## HEAD (Unreleased)

## 0.4.0

- use `gcr.io/distroless/static:lates` as base image ([#74](https://github.com/pingcap/advanced-statefulset/pull/74))
  - `tzdata` is packed into image, now it's ok to configure timezone via `TZ`
    env
  - shell utilities are removed, see [why](https://github.com/GoogleContainerTools/distroless#why-should-i-use-distroless-images)
