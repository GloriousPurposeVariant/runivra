FROM odoo:{{.Version}}

USER root

# Python packages for your custom addons: list them in custom/requirements.txt.
# The file is optional; the build works without it.
RUN --mount=type=bind,source=.,target=/tmp/project \
    if [ -f /tmp/project/custom/requirements.txt ]; then \
        pip3 install --break-system-packages --no-cache-dir -r /tmp/project/custom/requirements.txt; \
    fi

USER odoo
