# fops-router

`fops-router` est une image Docker basée sur **Caddy** (construite avec `xcaddy`) avec une configuration prête à l’emploi pour servir de **reverse-proxy de routage** dans une stack Docker.

Objectifs :
- centraliser les vhosts d’une stack (plusieurs services / plusieurs noms)
- fournir un **mode maintenance** pilotable via l’admin API Caddy sur socket Unix
- fournir des **journaux d’accès standardisés** (texte compatible “combined” + JSON) par vhost
- permettre des surcharges via variables d’environnement sans toucher au Caddyfile principal

## Ce que contient l’image

### Caddy + plugins
- `github.com/e-frogg/fops-caddy-maintenance` : directive `maintenance` + points d’accès d’admin API
- `github.com/e-frogg/fops-caddy-router` : points d’accès d’admin API pour pousser des vhosts dynamiques depuis fops-cli
- `github.com/caddyserver/transform-encoder` : utilisé pour formater le journal compatible “combined”
- `github.com/caddy-dns/ovh` : fournisseur DNS OVH pour les défis ACME DNS-01

### Outils inclus
- `bash`, `curl` (pratique pour le débogage et pour piloter l’admin API via socket Unix)

## Comment ça marche (architecture de configuration)

### 1) Caddyfile principal
Fichier : `/etc/caddy/Caddyfile` (fourni par l’image).

- expose l’admin API Caddy uniquement sur le socket Unix `/run/caddy/admin.sock`
- force l’ordre : `order maintenance first` (le middleware “maintenance” passe avant le reste)
- charge automatiquement tous les vhosts : `import /etc/fops-router/entrypoints/*.Caddyfile`
- charge les vhosts dynamiques générés : `import /data/fops-router/generated/*.Caddyfile`
- accepte des surcharges via variables :
  - `CADDY_MAIN_DIRECTIVES`, écrit au démarrage dans `/data/fops-router/runtime/main-directives.Caddyfile` puis importé dans le bloc global `{ ... }`
  - `CADDY_VHOSTS`, écrit au démarrage dans `/data/fops-router/runtime/vhosts.Caddyfile` puis importé après les imports

### 2) Extrait commun à importer dans chaque vhost
Fichier : `/etc/fops-router/entrypoint.Caddyfile` (fourni par l’image).

À importer **dans chaque bloc de site**, par exemple :
```caddyfile
example.localhost {
    import /etc/fops-router/entrypoint.Caddyfile app
    reverse_proxy app:8080
}
```

Cet extrait :
- installe le bloc `maintenance { ... }` (config via `CADDY_MAINTENANCE_DIRECTIVES`)
- insère `{$CADDY_SERVER_EXTRA_DIRECTIVES}` (directives communes à tous les vhosts qui importent l’extrait)
- configure 2 journaux d’accès par point d’entrée (le nom vient de l’argument `app` ci-dessus)

### 3) Tes vhosts / points d’entrée
Dossier : `/etc/fops-router/entrypoints/`

Tu y mets des fichiers `*.Caddyfile` qui contiennent tes blocs de site (un ou plusieurs par fichier).

### 4) Vhosts dynamiques
Dossier généré : `/data/fops-router/generated/`

Le plugin `fops-caddy-router` maintient :
- `/data/fops-router/registry.json` : registre persistant des stacks déclarées.
- `/data/fops-router/generated/routes.Caddyfile` : vhosts générés depuis le registre.

À chaque modification valide, le plugin régénère le Caddyfile dynamique complet puis recharge Caddy. Ce choix garde un état généré simple et déterministe ; les mutations de routes sont rares, donc le coût d’un reload complet est acceptable.

## Variables d’environnement (référence)

- `CADDY_MAIN_DIRECTIVES` : directives dans le bloc global.
  - Exemple :
    ```bash
    CADDY_MAIN_DIRECTIVES='email admin@example.com
debug'
    ```
- `CADDY_VHOSTS` : vhosts ajoutés en ligne (alternative aux fichiers `entrypoints/*.Caddyfile`).
- `CADDY_MAINTENANCE_DIRECTIVES` : sous-directives du bloc `maintenance { ... }`.
- `CADDY_SERVER_EXTRA_DIRECTIVES` : injecté dans chaque vhost qui importe `entrypoint.Caddyfile` (pratique pour `encode`, `header`, etc.).

Note : ces variables sont écrites telles quelles dans des fichiers `*.Caddyfile` runtime au démarrage du conteneur.

Exemple (appliquer des directives à tous les vhosts qui importent l’extrait) :
```yaml
environment:
  CADDY_SERVER_EXTRA_DIRECTIVES: |
    encode zstd gzip
    header {
      X-Content-Type-Options nosniff
    }
```

## Routage dynamique

Le routeur expose un wrapper métier interne au conteneur, appelé depuis l’hôte avec `docker exec`. `fops-cli` ne connaît que le nom du conteneur ; le wrapper masque l’admin API Caddy locale, le socket Unix `/run/caddy/admin.sock` et les endpoints HTTP internes.

Commandes :
- `fops-routerctl status`
- `fops-routerctl put-stack <project> <instance> <json-payload>`
- `fops-routerctl delete-stack <project> <instance>`

Exemple depuis l’hôte :
```bash
docker exec fops-router fops-routerctl put-stack example-app production '{
    "project": "example-app",
    "instance": "production",
    "routes": [
      {
        "id": "main",
        "hosts": ["app.example.com", "admin.example.com"],
        "entrypoint": "example-app-production-main",
        "upstream": {
          "scheme": "https",
          "network_alias": "example-app-caddy",
          "port": 443,
          "tls_server_name": "app.example.com",
          "tls_insecure_skip_verify": false
        }
      }
    ]
  }'
```

Le `PUT` remplace l’état complet de la stack `{project}/{instance}`. Supprimer une route côté fops-cli revient donc à ne plus l’envoyer au prochain `PUT`. Le `DELETE` retire toute la stack.

Attention : `tls_insecure_skip_verify` est strictement réservé au développement ou aux tests temporaires. En production, garde cette valeur à `false` et utilise des certificats valides côté upstream.

Le plugin refuse les collisions : un même hôte ne peut pas être déclaré par deux stacks différentes.

### Réseau Docker partagé

Les services amont dynamiques ciblent des alias Docker sur un réseau partagé. Convention recommandée :
```yaml
networks:
  fops-router:
    external: true

services:
  caddy:
    networks:
      fops-router:
        aliases:
          - example-app-caddy
```

Pour les stacks Symfony/FrankenPHP sans service `caddy` séparé, l’alias peut pointer vers le service frontal `php`.

### Déploiement rapide

Le fichier `compose.yaml` fourni est un exemple de déploiement production minimal.

```bash
cp .env.example .env
mkdir -p entrypoints
docker compose up -d
```

Points importants :
- renseigne `CADDY_ACME_EMAIL`
- connecte les services à router au réseau Docker `fops-router` ou déclare un alias réseau utilisé dans les charges utiles dynamiques
- ajoute tes vhosts statiques dans `entrypoints/*.Caddyfile` si tu n’utilises pas uniquement l’API dynamique

Exemple de point d’entrée statique :

```caddyfile
app.example.com {
    import /etc/fops-router/entrypoint.Caddyfile app
    reverse_proxy app:8080
}
```

## Mode maintenance (guide concret)

Le plugin expose des points d’accès sur l’admin API Caddy locale :
- statut : `GET /maintenance/status`
- activer/désactiver : `POST /maintenance/set` avec un JSON

Exemples :
```bash
# Statut
docker exec fops-router curl --unix-socket /run/caddy/admin.sock \
  http://localhost/maintenance/status

# Activer
docker exec fops-router curl --unix-socket /run/caddy/admin.sock \
  -X POST -H "Content-Type: application/json" \
  -d '{"enabled": true}' \
  http://localhost/maintenance/set

# Désactiver
docker exec fops-router curl --unix-socket /run/caddy/admin.sock \
  -X POST -H "Content-Type: application/json" \
  -d '{"enabled": false}' \
  http://localhost/maintenance/set
```

Configuration typique (via `CADDY_MAINTENANCE_DIRECTIVES`) :
```caddyfile
default_enabled false
allowed_ips_file /etc/fops-router/maintenance-ips.txt
retry_after 300
```

Le fichier d’état est fixé par l’image à : `/data/maintenance-status.json` (monte `/data` si tu veux persister l’état entre redémarrages).

Fichier d’IPs autorisées (`/etc/fops-router/maintenance-ips.txt`) :
- 1 IP ou CIDR par ligne
- commentaires possibles

Important (si tu es derrière un proxy ou un répartiteur de charge) :
- par défaut, le plugin utilise l’IP directe (`RemoteAddr`)
- si tu veux lire `X-Forwarded-For`/`X-Real-IP`, configure **explicitement** :
  ```caddyfile
  use_forwarded_headers true
  trusted_proxies 10.0.0.0/8 172.16.0.0/12 192.168.0.0/16
  ```
  (ne l’active pas si tu ne maîtrises pas les proxies devant Caddy)

## Journaux d’accès (par point d’entrée)

L’extrait écrit 2 fichiers par vhost (argument `import ... <name>`):
- `/var/log/caddy/<name>.access.log` : format texte compatible “combined”
  - rotation : 1 GiB, conservation ~13 mois
- `/var/log/caddy/<name>.access.json` : JSON
  - rotation : 100 MB, conserve 5 fichiers / ~30 jours

Conseil : monte `/var/log/caddy` en volume si tu veux conserver ou collecter ces fichiers.

## Exemple Docker Compose (stack)

Exemple (avec des fichiers de points d’entrée montés en volume) :
```yaml
services:
  fops-router:
    build:
      context: ./fops-router
      args:
        # Doit correspondre à un tag valide de l'image `caddy:<tag>`
        # (ex: "2.10.2")
        FOPS_IMAGE_VERSION: ${FOPS_IMAGE_VERSION}
    ports:
      - "443:443"
      - "443:443/udp"
    environment:
      CADDY_MAINTENANCE_DIRECTIVES: |
        default_enabled false
        allowed_ips_file /etc/fops-router/maintenance-ips.txt
    volumes:
      - ./entrypoints:/etc/fops-router/entrypoints:ro
      - ./maintenance-ips.txt:/etc/fops-router/maintenance-ips.txt:ro
      - caddy_data:/data
      - caddy_logs:/var/log/caddy

volumes:
  caddy_data:
  caddy_logs:
```

Variante (Docker Swarm / configs) : monter chaque point d’entrée comme une config.
```yaml
services:
  fops-router:
    configs:
      - source: fops-router-app-entrypoint
        target: /etc/fops-router/entrypoints/app.Caddyfile
      - source: fops-router-maintenance-ips
        target: /etc/fops-router/maintenance-ips.txt
```

Exemple de point d’entrée : `entrypoints/app.Caddyfile`
```caddyfile
app.localhost {
    import /etc/fops-router/entrypoint.Caddyfile app

    reverse_proxy app:8080 {
        header_up Host {host}
        header_up X-Real-IP {remote_host}
        header_up X-Original-URI {uri}
        header_up X-Original-Query {query}
        header_up X-Forwarded-Proto https
        header_up X-Forwarded-Port 443
    }
}
```

## Cas d’usages (rapide)

- Routage de stack (dev/staging/prod) : 1 service = 1 vhost = 1 fichier de journalisation (simple à maintenir).
- Maintenance pendant un déploiement : `POST /maintenance/set` + liste d’autorisation d’IP (équipe, VPN, bastion).
- Journaux exploitables : `*.access.log` (lecture humaine) + `*.access.json` (collecte/ingestion).
- Valeurs par défaut partagées : compression, en-têtes et délais d’expiration via `CADDY_SERVER_EXTRA_DIRECTIVES`.

## Sécurité (à ne pas zapper)

- L’admin API Caddy écoute par défaut sur le socket Unix réservé au propriétaire `/run/caddy/admin.sock`, pas en TCP.
- Le contrat d’administration à distance est `docker exec <container> fops-routerctl ...`; l’accès est donc contrôlé par les droits Docker sur l’hôte.
- Si tu modifies le Caddyfile pour réactiver une écoute TCP admin, garde-la strictement privée et filtrée.
