## System.d Service
Depro includes a System.d service configuration file for use on CentOS/RHEL/Fedora
systems. It assumes that you have a `consul.service` configured and will only
load once that service has started - if you do not wish to keep this behaviour,
simply remove `consul.service` from the Requires entry.

To enable the Depro service, simply do the following.

1. Copy the contents of the [depro.service] file into your `/etc/systemd/system`
directory.
1. Enable the service with `sudo systemctl enable depro.service`.
1. Start the service with `sudo systemctl start depro.service`
