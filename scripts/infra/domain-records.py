#!/usr/bin/env python3
"""Read App Platform's certificate challenges into an OpenTofu variable file."""
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import time
import urllib.request


def certificate_records(domains):
    records = {}
    for domain in domains:
        validations = (domain.get("validations") or []) + [domain.get("validation") or {}]
        for value in validations:
            name, token = value.get("txt_name"), value.get("txt_value")
            if name and token:
                key = hashlib.sha256(f"{name}:{token}".encode()).hexdigest()[:20]
                records[key] = {"name": name.rstrip("."), "value": token}
    return records


def main():
    root = Path(sys.argv[1]).resolve()
    wait = "--wait" in sys.argv[2:]
    output = subprocess.run(["tofu", f"-chdir={root}", "output", "-json"],
                            text=True, capture_output=True, check=True)
    saved = json.loads(output.stdout)
    previous = saved.get("domain_validation_records", {}).get("value", {})
    app_id = saved.get("app_id", {}).get("value")
    records = previous
    if app_id:
        for attempt in range(61 if wait else 1):
            request = urllib.request.Request(
                f"https://api.digitalocean.com/v2/apps/{app_id}",
                headers={"Authorization": "Bearer " + os.environ["DIGITALOCEAN_ACCESS_TOKEN"]})
            with urllib.request.urlopen(request, timeout=30) as response:
                domains = json.load(response)["app"].get("domains", [])
            discovered = certificate_records(domains)
            if discovered:
                records = discovered
                break
            if previous or all(d.get("phase") == "ACTIVE" for d in domains) and domains:
                break
            if not wait:
                break
            if attempt == 60:
                raise RuntimeError("App Platform has not issued wildcard verification records")
            time.sleep(5)
    path = root / "domain-validation.auto.tfvars.json"
    path.write_text(json.dumps({"domain_validation_records": records}, indent=2) + "\n")
    print(f"Prepared {len(records)} certificate DNS records")


if __name__ == "__main__":
    main()
