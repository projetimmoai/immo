# Directives du projet : Projet Immo AI (gestion immobilière)

Worker d'arrière-plan autonome en Go pour la surveillance des e-mails, l'extraction des données et l'insertion dans la base de données.

---

## 1. Règles de développement strictes
- **Arrêt propre (*Graceful Shutdown*)** : Toujours propager `context.Context` à toutes les fonctions I/O (appels réseau, requêtes SQL).
- **Gestion des erreurs** : 
  - Ne jamais ignorer d'erreur silencieusement.
  - Logger avec contexte : `log.Printf("Erreur %s: %v", msgID, err)`.
- **Base de données** :
  - Accès exclusivement via l'API REST de Supabase (PostgREST), avec la clé `service_role` (`internal/repository`, client `net/http` fait main). Pas de connexion Postgres directe (`pgx`/`database/sql`) depuis l'application — la connexion Postgres directe (`SUPABASE_DB_URL`) sert uniquement aux migrations de schéma, exécutées manuellement.
  - Pour tout traitement impliquant plusieurs écritures liées (ex: insérer un `email` et créer l'`incident` associé), utiliser une fonction Postgres (`plpgsql`) appelée en RPC (`/rest/v1/rpc/...`) plutôt qu'une transaction Go — l'API REST ne permet pas d'ouvrir une transaction à cheval sur plusieurs appels HTTP. La fonction elle-même est du SQL standard, portable hors Supabase.
  - Les ID des tables de référence (catégories, statuts...) ne sont jamais codés en dur : toujours résolus par une recherche sur `description`.
- **Montants monétaires** : Toujours stocker et manipuler les loyers et charges en entiers représentant les centimes (ex. `125000` pour `1 250,00 €`).

---

## 2. Politique de test obligatoire
- Avant de considérer une tâche terminée :
  1. Exécute `go test ./...` dans le terminal.
  2. Corrige immédiatement tout échec ou régression.
  3. Vérifie l'absence d'erreurs de lint avec `golangci-lint run`.
