# acbuild begin

`acbuild begin` will start a new build.

## Location of work context

By default, information about the build is stored at `.acbuild` in the current
working directory. If the current directory changes during the build, `acbuild`
will be unaware of, and unable to operate on, the build that was started until
the current directory is changed back to the location where `acbuild begin` was
run. If this is undesirable, the `--work-path` flag can be provided to specify
the location to store and access the build context.

## Starting state

The build will default to starting with an empty ACI. The rootfs will be empty,
and the manifest will look something like the following:

```json
{
    "acKind": "ImageManifest",
    "acVersion": "0.7.1+git",
    "name": "acbuild-unnamed",
    "labels": [
        {
            "name": "arch",
            "value": "amd64"
        },
        {
            "name": "os",
            "value": "linux"
        }
    ]
}
```

The `arch` and `os` labels are filled in with the architecture and operating
system of the machine acbuild is running on. If this is undesirable, the labels
can be modified or removed with the `acbuild label` command.

The begin command can also be passed an ACI, either on the file system or an
image name to fetch via [meta
discovery](https://github.com/appc/spec/blob/master/spec/discovery.md#meta-discovery).
When an ACI is specified, it is used as the starting point for the build as
opposed to an empty image. If the image is to be fetched via meta discovery
over http (as opposed to https), the `--insecure` flag must be used.

## Examples

```bash
acbuild begin
acbuild begin ./my-app.aci
acbuild begin quay.io/coreos/alpine-sh
acbuild --work-path /tmp/mybuild begin
```
