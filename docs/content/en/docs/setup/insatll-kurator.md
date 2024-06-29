---
title: "Install Kurator Cli"
linkTitle: "Install Kurator Cli"
weight: 10
description: >
  Instructions on installing Kurator cli.
---

## Install from source

Download the Kurator source and build the `kurator` executable. 
After building, the executable will be located in the `./out/{your_os}` directory, such as `./out/linux-amd64` for Linux amd64 systems. 
You need to move it to a directory included in your system's PATH. 
The PATH is an environment variable indicating where the system can find executable files, for example, `/usr/local/bin`.

```bash
git clone https://github.com/kurator-dev/kurator.git
cd kurator
make kurator
sudo mv ./out/linux-amd64/kurator /usr/local/bin/
```

## Install from release package

To install from a release package, first navigate to the [Kurator release](https://github.com/kurator-dev/kurator/releases) page. 
Download the package suitable for your operating system and extract it directly to a directory in your PATH.

Below is a command-line example for amd64 architectures, targeting the PATH `/usr/local/bin`:

```console
curl -LO https://github.com/kurator-dev/kurator/releases/download/v{{< kurator-version >}}/kurator-{{< kurator-version >}}-linux-amd64.tar.gz
sudo tar -zxvf kurator-{{< kurator-version >}}-linux-amd64.tar.gz -C /usr/local/bin/
```

## Verify Installation

After installing Kurator, you can verify the installation by checking the version of the installed software. 
To do this, run the `kurator version` command in your terminal. 
This command should return information about the Kurator version you have installed, similar to the following output:

```bash
kurator version
```

You should see output resembling:

```json
{
  "gitVersion": "0.5.0",
  "gitCommit": "b964c81e22bf68fa9eb02ab4c6a4bc887ef620b7",
  "gitTreeState": "clean",
  "buildDate": "2023-10-30T12:49:17Z",
  "goVersion": "go1.20.2",
  "compiler": "gc",
  "platform": "linux/amd64"
}
```

## Playground

Kurator uses killercoda to provide [Kurator install demo](https://killercoda.com/965010e0-4f60-4a28-bf27-597d3kurator/scenario/install-kurator), allowing users to experience hands-on operations.
