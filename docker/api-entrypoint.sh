#!/bin/sh
set -eu

case "${ASTRO_RUNTIME_ENV:-}" in
    test|production)
        source_kubeconfig=${ASTRO_KUBERNETES_KUBECONFIG:-}
        if [ "$source_kubeconfig" != "/run/secrets/astro-kubeconfig" ] || [ ! -r "$source_kubeconfig" ] || [ ! -s "$source_kubeconfig" ]; then
            printf '%s\n' '运行时 kubeconfig 挂载不可用' >&2
            exit 1
        fi

        target_kubeconfig=/run/astro/kubeconfig
        cp "$source_kubeconfig" "$target_kubeconfig"
        chown astro:astro "$target_kubeconfig"
        chmod 0400 "$target_kubeconfig"
        export ASTRO_KUBERNETES_KUBECONFIG=$target_kubeconfig
        ;;
    *)
        printf '%s\n' 'API 容器仅允许 ASTRO_RUNTIME_ENV=test 或 production' >&2
        exit 1
        ;;
esac

exec su-exec astro:astro "$@"
