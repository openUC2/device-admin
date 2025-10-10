# device-admin
A system settings panel for machine administration

## Introduction

[ImSwitch OS](https://github.com/openuc2/imswitch-os) (which runs on a Raspberry Pi computer)
provides the Cockpit system administration panel, but that panel is behind a login screen and is
missing various functionalities. This tool provides a web browser interface for other
functionalities needed by customers who operate openUC2 instruments, such as:

- Wi-Fi network conntion management (which relies on NetworkManager)
- Toggling remote assistance (which relies on Tailscale)
- Software updates (which uses Forklift)

It is meant to be served from a reverse-proxy on port 80 along with all other network
services, configured as in [openUC2/pallet](https://github.com/openUC2/pallet).

In the future, this tool will probably be extended to give the user (or otherwise direct the user to) a
setup wizard for configuring localization settings (e.g. for languages and wifi networks) upon the
first boot of the OS.

## Usage

### Local Deployment

First, you will need to download device-admin, which is available as a single self-contained
executable file. You should visit this repository's
[releases page](https://github.com/openUC2/device-admin/releases/latest) and download an archive
file for your platform and CPU architecture; for example, on a Raspberry Pi 5, you should download
the archive named `device-admin_{version number}_linux_arm.tar.gz` (where the version number should
be substituted). You can extract the device-admin binary from the archive using a command like:
```
tar -xzf device-admin_{version number}_{os}_{cpu architecture}.tar.gz device-admin
```

Then you may need to move the device-admin binary into a directory in your system path, or you can just run the device-admin binary in your current directory (in which case you should replace `device-admin` with `./device-admin` in the commands listed below).

Once you have device-admin, you can run it as follows on a Raspberry Pi:
```
./device-admin
```

Then you can view the landing page at <http://localhost:3001> . Note that if you are running it on a
computer other than the Raspberry Pi with ImSwitch OS, then you will need to set some environment
variables (see below) to non-default values.

### Development

To install various backend development tools, run `make install`. You will need to have installed Go first.

Before you start the server for the first time, you'll need to generate the webapp build artifacts by running `make buildweb` (which requires you to have first installed [Node.js](https://nodejs.org/en/) and [Yarn Classic](https://classic.yarnpkg.com/lang/en/)). Then you can start the server by running `make run` with the appropriate environment variables (see below); or you can run `make runlive` so that your edits to template files will be reflected after you refresh the corresponding pages in your web browser. You will need to have installed golang first. Any time you modify the webapp files (in the web/app directory), you'll need to run `make buildweb` again to rebuild the bundled CSS and JS.

### Building

Because the build pipeline builds Docker images, you will need to either have Docker Desktop or (on Ubuntu) to have installed QEMU (either with qemu-user-static from apt or by running [tonistiigi/binfmt](https://hub.docker.com/r/tonistiigi/binfmt)). You will need a version of Docker with buildx support.

To execute the full build pipeline, run `make`; to build the docker images, run `make buildall`. Note that `make buildall` will also automatically regenerate the webapp build artifacts, which means you also need to have first installed Node.js as described in the "Development" section. The resulting built binaries can be found in directories within the dist directory corresponding to OS and CPU architecture (e.g. `./dist/device-admin_window_amd64/device-admin.exe` or `./dist/device-admin_linux_amd64/device-admin`)

### Environment Variables

#### Custom Templates

You can override the default webpage templates embedded in the device-admin binary by providing a path to the templates directory with the `TEMPLATES_PATH` variable, relative to the current working directory in which you start the device-admin program. For example, you could provide a custom home page by creating a new file named `home.page.tmpl` with following contents in a new `custom-templates/home` subdirectory in the directory from which you will launch device-admin:
```
{{template "shared/base.layout.tmpl" .}}

{{define "title" -}}
  Machine administration
{{- end}}
{{define "description"}}Machine system settings{{end}}

{{define "content"}}
  <main>
    <section class="section content">
      <div class="container">
        <h1>Hello, world!</h1>
        <p>
          Greetings from a custom template!
        </p>
    </section>
  </main>
{{end}}
```

and then running the following command:
```
# If you downloaded a device-admin binary:
TEMPLATES_PATH=custom-templates ./device-admin
# If you are developing the project:
TEMPLATES_PATH=custom-templates make run
```

## Licensing

Except where otherwise indicated, source code provided here is covered by the following information:

Copyright Ethan Li and openUC2 project contributors

SPDX-License-Identifier: `Apache-2.0 OR BlueOak-1.0.0`

You can use the source code provided here either under the [Apache 2.0 License](https://www.apache.org/licenses/LICENSE-2.0) or under the [Blue Oak Model License 1.0.0](https://blueoakcouncil.org/license/1.0.0); you get to decide. We are making the software available under the Apache license because it's [OSI-approved](https://writing.kemitchell.com/2019/05/05/Rely-on-OSI.html), but we like the Blue Oak Model License more because it's easier to read and understand.
