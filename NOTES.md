# Notes

Notes de travail libres — étapes à venir, idées, rappels. Pas un journal de
décisions techniques (ça, c'est dans les commits et le code) : plutôt un
pense-bête pour ne rien perdre entre deux sessions.

## Prochaines étapes

- `internal/email` : décider IMAP vs Gmail API pour la connexion à la boîte mail, puis implémenter la surveillance.
- Connecter au Drive (compte de service Google). Code déjà écrit : `internal/drive` (client REST Drive v3), `internal/service.CoproprieteService.CreateCopropriete` (crée la copropriete en base + son arborescence Drive `<reference> - <nom>/{contrats,sinistres,incidents,emails,assemblees_generales}`, idempotent côté Drive). Reste à faire : créer le compte de service Google (Drive API activée), le partager en accès sur le dossier `NE PAS MODIFIER - fichiers`, renseigner `GOOGLE_SERVICE_ACCOUNT_JSON` et `DRIVE_ROOT_FOLDER_ID` dans `.env` (voir `.env.example`). Une fois fait, `internal/email` pourra aussi s'en servir pour les pièces jointes (`email_piece_jointe.url`).

## Idées / à explorer plus tard

- Assemblée générale : modéliser `assemblee_generale` (table absente pour l'instant, `exercice_comptable.ag_approbation_id` en attente).
- Renforcer l'intégrité référentielle restante si besoin (cf. discussion sur les FK).
- `internal/check.CheckCoproprietes` : vérifie la cohérence base/Drive, scope limité aux dossiers de copropriété pour l'instant. À étendre au fur et à mesure (nouvelles tables/dossiers Drive). Pas encore branché sur un cron — à faire quand `internal/email`/le worker seront en place (le cron pourra être un ticker dans `cmd/worker-email`, ou un endpoint dédié appelé par un scheduler externe).

## Vrac
