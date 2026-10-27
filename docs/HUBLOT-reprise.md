# Hublot : dossier de reprise complet

> Document unique pour reprendre le projet ailleurs, avec un autre agent.
> Porteur : ET Technology Consulting (Enzo Tripoli). Premier utilisateur : NovaMoon.
> Rédigé le 2026-10-26 à partir des échanges des 25 et 26 octobre 2026.
> « Hublot » est un nom de code, pas encore définitif.

## Sommaire

1. Comment utiliser ce document (consignes pour l'agent qui reprend)
2. Le projet en une page
3. Décisions prises, et pourquoi
4. Ce qui reste ouvert
5. Identité visuelle « Franc »
6. Le concept de marque : le hublot
7. Inventaire des maquettes
8. Landing page (FR et EN)
9. Prochaines étapes proposées
10. PRD complet (annexe A)
11. Références de l'API GitHub (dans le PRD)

---

## 1. Consignes pour l'agent qui reprend

- **Lis ce document en entier avant d'écrire du code.** Les décisions de la section 3 ne se rediscutent pas sans Enzo.
- **Le projet vit dans son propre dépôt**, sous l'organisation GitHub d'ET Technology Consulting. Il n'a rien à voir avec le dépôt `media-poste-ads` (NovaMoon Ads), dans lequel ces échanges ont eu lieu.
- **GitHub est la seule source de vérité.** Toute fonctionnalité qui stockerait du travail hors de GitHub est à refuser (principe 4 du PRD).
- **Ne teste pas et n'étudie pas les outils concurrents** (Quackback, Canny, etc.) : Enzo veut concevoir sans biais, et le projet est développé de façon indépendante (propriété intellectuelle nette pour le CLA).
- **Pas de style « généré par IA »** : pas de dégradés, pas de bleu par défaut, pas de cartes fades. Suis la direction Franc (section 5).
- **Conversation en français ; code, commits et noms techniques en anglais**, commits au format Conventional Commits.
- **Licence AGPL-3.0**, droits au nom d'ET Technology Consulting, **CLA obligatoire** dès la première contribution externe.
- **Les sources des maquettes** sont dans l'archive `hublot-maquettes.zip` livrée avec ce document (fichiers `.dc.html`, un par écran, plus `canvas.json` qui donne leur disposition). Le canevas en ligne d'origine est https://claude.ai/artifact/AZxi4eSp1sKVLwXqpzb483 (privé, compte d'Enzo).

## 2. Le projet en une page

**Le problème.** Une petite structure qui fait plusieurs logiciels empile les outils (tracker, boîte à idées, helpdesk, roadmap, changelog) alors que GitHub sait déjà presque tout faire. Ce qui manque : une porte d'entrée pour les gens qui n'ont pas de compte GitHub et n'ont pas à voir le code.

**La solution.** Un portail open source posé sur GitHub :
- l'équipe non technique **dépose** une demande (bug, idée, question) de façon guidée, avec captures, sans compte GitHub (lien magique par e-mail) ;
- elle **suit** chaque demande comme un colis (Reçue → Planifiée → En réalisation → En test → Disponible), le « Disponible » étant lu dans les déploiements GitHub ;
- elle **voit ce qui arrive** (roadmap Maintenant / Bientôt / Plus tard) et les **nouveautés** en mots simples ;
- l'équipe produit **trie** toutes les demandes de tous les dépôts au même endroit, **crée des fonctionnalités et leurs issues de façon guidée** (jauge « prête »), et **gère les accès par logiciel** (RBAC) ;
- **Claude** peut tout créer depuis un prompt grâce à un **serveur MCP intégré**.

**Qui l'utilise.** Les développeurs (Enzo, Claude) restent dans GitHub ; Enzo utilise aussi le portail pour présenter et trier ; l'équipe non tech n'utilise que le portail.

**Stack.** Go (chi, sqlc, goose, River, go-github, githubv4, ghinstallation) et React/Vite/TypeScript embarqué dans un seul binaire ; PostgreSQL ; stockage S3 ; une seule image Docker. Détail en annexe A, §9.

## 3. Décisions prises

| Date | Décision | Pourquoi |
|---|---|---|
| 25/10 | Garder GitHub comme outil des devs ; un portail à part pour les non-tech | Réduire le nombre d'outils ; GitHub est complet mais sous-utilisé |
| 25/10 | GitHub reste la **seule source de vérité** ; le portail ne stocke que comptes, pièces jointes, cache, liens | Supprimer le portail ne fait rien perdre ; pas de modèle parallèle |
| 25/10 | Projet **open source**, porté par **ET Technology Consulting** | Combler un vrai manque ; visibilité |
| 25/10 | Licence **AGPL-3.0** + **CLA** dès la 1re contribution externe | Protéger une éventuelle offre hébergée ; garder la double licence possible |
| 25/10 | Ne pas tester Quackback ni les concurrents | Concevoir sans biais |
| 25/10 | **RBAC** par logiciel, refus par défaut ; rôles viewer, reporter, contributor, triager, manager ; groupes | Les non-tech ne voient que leurs logiciels |
| 25/10 | Réglage « les reporters voient toutes les demandes » **désactivé par défaut** | On ouvre plus facilement qu'on ne referme |
| 25/10 | Création **guidée** des fonctionnalités et issues, champs obligatoires, **définition du « prêt »**, gabarit Markdown versionné `<!-- hublot:issue v1 -->` | Exploitable par Claude, compréhensible par Enzo |
| 25/10 | **Serveur MCP intégré** (`/mcp`, OAuth ou jeton, droits de la personne, brouillon par défaut, `create_feature` avec ses issues) + écran de configuration | Créer fonctionnalités et issues depuis un prompt Claude |
| 25/10 | Direction visuelle **Franc** retenue (section 5) | Ne fait pas « tech » ni « IA générique » |
| 25/10 | Landing page validée | |
| 25/10 | **Hors V1, pour plus tard** : clé API Claude de l'organisation pour pré-remplir les champs à partir du code ; déclencheurs entrants (e-mail, formulaire, webhook) pré-associés et pré-décrits | Noté pour les versions suivantes |
| 26/10 | Concept de marque **hublot** : un même hublot sur un véhicule par projet, une **fusée** pour l'entreprise ; le hublot seul est le logo | Rester sur le sujet « hublot » ; clin d'œil au fils d'Enzo (fan d'avions) avec l'avion parmi les véhicules |
| 26/10 | Haut de la landing = « Le voyage », noms de projets **génériques** ; landing en **français et anglais** | |

## 4. Ce qui reste ouvert

- **Le nom définitif** du projet (« Hublot » est provisoire, mais le concept de marque est construit dessus).
- Le texte du **CLA** (outil conseillé : CLA Assistant).
- Dépôt de retours commun ou dépôt direct dans chaque logiciel.
- Règle de publication des commentaires GitHub vers le portail (préfixe, label, ou réponse depuis le portail seulement).
- Hébergement de l'instance NovaMoon (OVH, par cohérence avec l'existant ?).
- Offre hébergée payante un jour ? (FAQ de la landing : « [À DÉCIDER] »).
- Choix du véhicule de chaque logiciel dans les réglages (le concept existe, l'écran Réglages ne le propose pas encore).

## 5. Identité visuelle « Franc »

### Jetons

| Rôle | Valeur |
|---|---|
| Fond de page | `#F4F1EC` (blanc cassé chaud) |
| Surfaces | `#FFFFFF` |
| Encre (texte, contours) | `#111111` |
| Texte secondaire | `#3A3A3A` (sur blanc) |
| Action principale, accent | jaune `#FFE066` |
| Couleurs de logiciel | lavande `#B9AEFF`, orange `#FF9A62`, menthe `#7FD8B6`, jaune `#FFE066`, rose `#FFB3D1` |
| Ciel (illustrations) | `#A8DBFF` ; mer `#6FB6E8` |
| Bague du hublot | `#ECE7DD` |
| Terminaux, blocs code | anthracite `#2E3036`, séparateur `#4A4D55` (jamais de noir sur ombre noire) |
| Bandeau d'alerte doux | `#FFF6CC` |

### Typographie

- Titres : **Bricolage Grotesque** 800 (600 pour les intermédiaires), interlettrage serré (-0,02 à -0,04 em), interlignage 0,92 à 1.
- Texte : **Atkinson Hyperlegible** 400 et 700 (conçue pour la lisibilité).
- Code et références : **IBM Plex Mono** 500.

### Formes

- **Contours noirs épais** : 2,5 px pour les éléments courants, 3 px pour les éléments mis en avant.
- **Ombres décalées franches**, sans flou : `4px 4px 0 #111111` à `8px 8px 0 #111111`, réservées aux actions principales et aux éléments mis en avant.
- Rayons : 10 à 22 px ; pastilles en 999 px.
- Bouton principal : fond jaune, contour 3 px, ombre décalée. Bouton secondaire : fond noir texte blanc, ou fond blanc contour noir.
- L'élément en cours (étape, onglet) passe en jaune avec ombre ; l'élément terminé en noir avec coche menthe ; l'élément à venir en contour pointillé.

### Ton et vocabulaire

- Familier et direct (« Salut Marie, on fait quoi aujourd'hui ? », « Moi aussi ! », « C'est envoyé ! »).
- Jamais de jargon : **demande** (pas issue/ticket), **logiciel** (pas repo), **disponible** (pas merge/deploy), « pris en charge par Enzo » (pas assignee), pas de numéro mis en avant.
- Accessibilité AA minimum, vrais boutons et liens, navigation au clavier.

### Composants et écrans clés

- **En-tête partagé** (composant unique) : logo, navigation équipe (Accueil, Mes demandes, Ce qui arrive, Nouveautés), séparateur, puis pour les admins À trier (avec compteur), Fonctionnalités, Accès, Réglages ; bouton « + Demande » ; avatar.
- **Suivi façon colis** : frise de 5 étapes, l'étape en cours en relief.
- **Jauge « prête »** et liste de contrôle à côté de toute création guidée.

## 6. Le concept de marque : le hublot

- **Un seul hublot**, identique partout : bague épaisse `#ECE7DD`, contour noir, **huit rivets**, vitre ronde avec un petit reflet blanc. La couleur de la vitre est celle du logiciel.
- **Chaque logiciel a un véhicule** (et sa couleur) : bateau, sous-marin, train, avion… Le véhicule remplace la pastille de couleur dans l'application.
- **La fusée qui décolle représente l'entreprise.** Elle porte le plus grand hublot.
- **Le logo est le hublot seul**, vitre jaune ; lisible à 16 px (favicon) et sur fond sombre.
- **L'avion reste parmi les véhicules** : c'est le clin d'œil au fils d'Enzo, fan d'avions.
- Le symbole SVG de référence (à réutiliser tel quel) :

```svg
<symbol id="hub" viewBox="-20 -20 40 40">
  <circle r="18" fill="#ECE7DD" stroke="#111111" stroke-width="2.5"/>
  <circle r="11" fill="currentColor" stroke="#111111" stroke-width="2.5"/>
  <path d="M-6-4a7 7 0 016-4" stroke="#FFFFFF" stroke-width="2" stroke-linecap="round" fill="none"/>
  <g fill="#111111">
    <circle cx="0" cy="-14.5" r="1.6"/><circle cx="0" cy="14.5" r="1.6"/>
    <circle cx="-14.5" cy="0" r="1.6"/><circle cx="14.5" cy="0" r="1.6"/>
    <circle cx="10.3" cy="-10.3" r="1.6"/><circle cx="-10.3" cy="10.3" r="1.6"/>
    <circle cx="-10.3" cy="-10.3" r="1.6"/><circle cx="10.3" cy="10.3" r="1.6"/>
  </g>
</symbol>
<!-- usage : <use href="#hub" width="40" height="40" style="color:#B9AEFF"/> -->
```

## 7. Inventaire des maquettes

Toutes en style Franc sauf indication. Les fichiers sont dans `hublot-maquettes.zip` (dossier `project/`). Les écrans « interactifs » réagissent aux clics en mode Play sur le canevas.

**Équipe (non tech)**

| Fichier | Écran |
|---|---|
| `C-Connexion.dc.html` | Connexion par lien magique (+ Google Workspace) |
| `C-LienEnvoye.dc.html` | Lien envoyé (mobile) |
| `C-Accueil.dc.html` | Accueil : actions rapides, compteurs, logiciels, « Ça bouge », Tout neuf, Bientôt |
| `C-NouvelleDemande.dc.html` | Nouvelle demande, web, 4 étapes, **interactif** (doublon « C'est aussi mon cas ») |
| `C-Depot.dc.html`, `C-Depot2..4.dc.html` | Nouvelle demande, mobile, étapes 1 à 4 |
| `C-Demandes.dc.html` | Mes demandes (onglets, recherche, badge confidentielle) |
| `C-Suivi.dc.html`, `C-SuiviMobile.dc.html` | Suivi d'une demande façon colis + échanges |
| `C-Roadmap.dc.html` | Ce qui arrive : Maintenant / Bientôt / Plus tard, « Moi aussi » |
| `C-Nouveautes.dc.html` | Nouveautés (« Tout neuf ») |
| `C-Email.dc.html` | E-mail de notification |

**Équipe produit (admin)**

| Fichier | Écran |
|---|---|
| `C-Triage.dc.html` | À trier : classer, transférer, doublon, titre public proposé |
| `C-Features.dc.html` | Liste des fonctionnalités (progression, personnes qui attendent) |
| `C-Feature.dc.html` | Nouvelle fonctionnalité et ses issues, **interactif**, complétude par issue |
| `C-Issue.dc.html` | Détailler une issue, guidé en 5 étapes, jauge « prête », aperçu du Markdown GitHub, **interactif** |
| `C-Acces.dc.html` | Personnes et accès (rôles par logiciel, groupes, héritage) |
| `C-Reglages.dc.html` | Réglages des logiciels (couleur, dépôt, visibilité) |
| `C-Mcp.dc.html` | Réglages › Claude et MCP, **interactif** (outils, mode brouillon, jetons, journal) |
| `Header.dc.html` | Composant d'en-tête partagé (importé par tous les écrans desktop) |

**Marque et site**

| Fichier | Contenu |
|---|---|
| `Site.dc.html` | Landing page française (validée) |
| `Site-EN.dc.html` | Landing page anglaise |
| `Voyage.dc.html` | « Le voyage » : un même hublot, un véhicule par projet, la fusée pour l'entreprise (haut de landing retenu) |
| `Emblemes.dc.html` | Icônes par véhicule, logo hublot, usage dans l'app |

**Explorations écartées** (gardées pour mémoire) : directions A « Éditorial » et B « Messagerie » (`Main`, `A-*`, `B-*`) ; quatre hauts de page avion (`Hero-*`) ; douze pistes visuelles (`Pistes-1`, `Pistes-2`) jugées trop éloignées du sujet « hublot ».

## 8. Landing page

- Structure : en-tête (logo hublot, navigation, sélecteur FR/EN, GitHub, Installer) → **Le voyage** → « Un outil de moins, pas un de plus » → Comment ça marche (3 étapes) → Pour l'équipe / Pour les développeurs → Ce que vous avez dedans (6 fonctions) → GitHub reste la source de vérité → **Avec Claude** (serveur MCP) → Auto-hébergement (terminal anthracite) → Libre pour de vrai / Né d'un vrai besoin → FAQ → pied de page.
- Accroche FR : « Chaque projet voyage à sa façon. Tout le monde regarde par le même hublot. »
- Accroche EN : « Every project travels its own way. Everyone looks through the same porthole. »
- Noms de projets d'exemple (génériques) : Boutique en ligne / Online shop, App mobile / Mobile app, Outil interne / Internal tool, Site vitrine / Website, Votre entreprise / Your company.
- Les deux langues sont deux fichiers distincts : **toute modification de texte se fait dans les deux**.
- À ne pas affirmer tant que ce n'est pas vrai : aucun chiffre d'usage, aucune référence client.

## 9. Prochaines étapes proposées

1. Créer le dépôt sous ET Technology Consulting avec ce document, l'archive des maquettes, la licence AGPL-3.0, `CONTRIBUTING.md` (avec le CLA) et `SECURITY.md`.
2. Configurer l'organisation GitHub de test : types d'issue, champs d'organisation (Priorité, Effort, Dates, Titre public), labels communs, formulaires d'issue (`feature.yml`, `task.yml`, `bug.yml`) reprenant les champs de la création guidée.
3. Créer la GitHub App (droits minimaux listés en annexe A, §8).
4. Jalon 1 : squelette Go + React, lien magique, RBAC et sa suite de tests d'isolation, dépôt d'une demande avec pièce jointe.
5. Jalon 2 : webhooks et cache, Mes demandes, triage, notifications.
6. Jalon 3 : création guidée et serveur MCP.
7. Usage interne chez NovaMoon, puis publication open source.

---

# Annexe A : PRD complet


> Nom de code provisoire : **Hublot** (une fenêtre sur GitHub). À remplacer.
> Porteur : ET Technology Consulting. Premier client : NovaMoon.
> Statut : brouillon v0.1, 2026-10-25.

---

## 1. Contexte et problème

Une petite structure qui développe plusieurs logiciels finit avec quinze outils : un tracker (Linear, Jira), un outil de retours (Canny), un helpdesk, une roadmap publique, un changelog, un wiki. Chacun duplique une partie de ce que GitHub sait déjà faire, et chacun est un abonnement, un compte et une synchronisation de plus.

GitHub couvre aujourd'hui presque tout le modèle d'un tracker moderne (types d'issue, champs personnalisés d'organisation, sous-issues, dépendances, Projects, releases, déploiements). Ce qui manque, c'est **une porte d'entrée pour les personnes qui n'ont pas de compte GitHub** et n'ont pas à voir le code : l'équipe métier, le support, la direction, parfois les clients.

Aucun outil existant ne fait ça en gardant GitHub comme source de vérité (voir §11).

## 2. Vision

> Toute l'équipe travaille sur GitHub, même ceux qui ne l'ouvriront jamais.

Les développeurs restent dans GitHub et `gh`. Le portail est une interface claire posée dessus, qui permet à n'importe qui de **déposer**, **suivre** et **comprendre** le travail de tous les logiciels de l'organisation, sans compte GitHub.

## 3. Principes non négociables

1. **GitHub est la seule source de vérité.** Statut, priorité, dates, hiérarchie, dépendances, versions vivent dans GitHub. Supprimer le portail ne fait perdre aucune information de travail.
2. **Le portail ne stocke que ce que GitHub ne sait pas porter** : les utilisateurs du portail, les pièces jointes, un cache, et le lien personne ↔ demandes.
3. **Configuration, jamais code en dur.** Logiciels, statuts, libellés, domaines autorisés, champs visibles : tout est configurable. Aucune trace d'un client particulier dans le code.
4. **Pas de grignotage.** Une fonction qui obligerait à stocker du travail hors de GitHub est refusée. On ne refait pas Linear.
5. **Simple à héberger** : un conteneur, une base Postgres, un stockage objet compatible S3.

## 4. Utilisateurs

| Persona | Où il travaille | Ce qu'il attend du portail |
|---|---|---|
| **Développeur** | GitHub, `gh`, agents IA | Rien, sinon que les demandes arrivent propres et bien rangées |
| **Responsable produit / tech lead** | GitHub **et** portail | Trier toutes les demandes au même endroit ; présenter la roadmap sans ouvrir GitHub |
| **Membre non tech** (support, ops, direction) | Portail uniquement | Signaler un bug ou une idée en deux minutes ; savoir où en est sa demande en mots simples |
| **Client invité** (V3, optionnel) | Portail, accès restreint | Voir la roadmap publique de son logiciel ; déposer une demande |

## 5. Périmètre

### V1 : déposer, suivre, trier

- **Connexion sans compte GitHub** : lien magique par e-mail, limité aux domaines autorisés. Option OIDC (Google Workspace, Microsoft) en configuration.
- **Contrôle d'accès par logiciel (RBAC)**, voir §5 bis : chacun ne voit et n'agit que sur les logiciels qui lui sont attribués, directement ou par un groupe.
- **Dépôt multi-logiciels** : choix du logiciel, puis le formulaire du dépôt cible. **Les formulaires sont ceux de GitHub** : le portail lit les fichiers YAML `.github/ISSUE_TEMPLATE/*.yml` et les affiche. Modifier un formulaire = modifier le dépôt.
- **Pièces jointes** : captures et fichiers stockés dans le bucket du portail, liés dans le corps de l'issue par URL signée ou proxy authentifié.
- **Suivi** : « Mes demandes » et « Toutes les demandes », filtrables par logiciel, type, statut. Statut traduit en mots simples (configurable).
- **Triage** (admin) : toutes les demandes entrantes de tous les dépôts dans une boîte unique ; classer en un clic (type, champ Priorité, logiciel) ; transférer vers le bon dépôt.
- **Notifications** : e-mail quand le statut d'une demande change ; résumé quotidien optionnel.
- **i18n** dès le départ : français et anglais.

### V2 : la vitrine

- **Roadmap** par logiciel et globale : features (type d'issue) avec la progression de leurs sous-issues, champs Start date / Target date, itérations des Projects.
- **Cycle de vie complet** : Reçue → Planifiée → En cours de dev (PR liée) → En revue → **Livrée en test** → **Livrée en prod** (lu depuis les Deployments/Environments).
- **Bloqué par** : affichage des dépendances.
- **Notes de version** lisibles, depuis les Releases.
- **Échanges dans les deux sens** : commentaires visibles côté portail selon une règle (préfixe ou label), réponse de l'utilisateur renvoyée en commentaire GitHub signé par l'app au nom de la personne.
- **Votes** sur les idées (réactions 👍 de l'issue, comptées par le portail pour les utilisateurs sans compte).

### V3 : ouverture

- Accès invités (clients externes) restreint à un logiciel.
- Roadmap publique en lecture seule, sans connexion.
- Widget intégrable dans une application (« Signaler un problème »).
- Recherche de doublons à la saisie.

### Versions suivantes (notées le 2026-10-25, pas en V1)

- **Pré-remplissage par l'IA avec la clé de l'organisation** : un admin ajoute une clé API Claude (stockée dans le coffre, jamais en clair). Lors de la création d'une fonctionnalité ou d'une issue, le portail propose les champs à partir de ce qui est saisi **et de la base de code** (lecture du dépôt via la GitHub App : zones concernées, critères plausibles, façon de vérifier). Toujours une proposition validée par un humain, jamais une écriture silencieuse.
- **Déclencheurs entrants** : comme le MCP mais dans l'autre sens. Un e-mail reçu sur une adresse dédiée, un formulaire de support, un webhook (outil de support, formulaire du site, monitoring) arrivent dans le triage, **pré-associés** au bon logiciel et au bon type par des règles de correspondance configurables, et **pré-décrits** (résumé, champs du gabarit remplis, doublon probable signalé). Ils restent des demandes à trier : rien n'entre dans GitHub sans qu'une personne l'ait classé.

### Hors périmètre

- Remplacer GitHub pour les développeurs.
- Gérer du travail qui n'existe pas dans GitHub.
- Helpdesk complet (SLA, macros, chat en direct).
- Support d'autres forges (GitLab, Gitea) avant que le cœur soit stable. L'architecture le permet (§8) mais ce n'est pas un objectif.

## 5 bis. Contrôle d'accès (RBAC)

Le portail a des droits larges sur GitHub ; ses utilisateurs n'en ont que ceux qu'on leur donne. **Refus par défaut** : un utilisateur sans attribution ne voit aucun logiciel.

### Deux niveaux de rôle

**Rôle dans l'organisation** (un par utilisateur) :

| Rôle | Peut |
|---|---|
| `owner` | Tout, y compris la GitHub App, la configuration et la gestion des autres owners |
| `admin` | Gérer les utilisateurs, les groupes et les attributions ; configurer les logiciels |
| `member` | Rien au niveau organisation ; ses droits viennent de ses rôles par logiciel |

**Rôle par logiciel** (attribué à une personne ou à un groupe) :

| Rôle | Voir les demandes | Déposer | Commenter | Voir la roadmap | Trier, prioriser, transférer | Gérer les accès du logiciel |
|---|---|---|---|---|---|---|
| `viewer` | Toutes | Non | Non | Oui | Non | Non |
| `reporter` | **Les siennes** (+ toutes si le logiciel l'autorise) | Oui | Sur les siennes | Oui | Non | Non |
| `contributor` | Toutes | Oui | Oui | Oui | Non | Non |
| `triager` | Toutes | Oui | Oui | Oui | Oui | Non |
| `manager` | Toutes | Oui | Oui | Oui | Oui | Oui |

- Un réglage par logiciel, **« Les reporters voient toutes les demandes »**, couvre les deux cultures : support qui ne doit voir que ses tickets, ou équipe où tout est transparent.
- Les droits se cumulent : le rôle effectif sur un logiciel est le plus élevé entre l'attribution directe et celles des groupes.
- Un `owner` ou `admin` voit tout, mais ses consultations restent journalisées.

### Groupes

- Des **groupes** (« Support », « Direction », « Client Allamau ») portent des attributions : ajouter quelqu'un au groupe lui donne tous les accès du groupe.
- V3 : groupes alimentés automatiquement par l'annuaire via OIDC (groupes Google Workspace ou Entra ID), pour qu'un départ retire les accès sans action manuelle.

### Visibilité au niveau de la demande

- Une demande peut être marquée **confidentielle** (label GitHub `portal:private`) : seuls son auteur, les `triager`/`manager` du logiciel et les admins la voient.
- Les **commentaires internes** (sans le préfixe ou label de publication) ne sont jamais visibles des `viewer`, `reporter` ni `contributor`.
- Une demande **transférée** vers un autre logiciel suit les droits du nouveau logiciel ; son auteur la voit toujours.

### Règles d'application

- Tout est appliqué **côté serveur, à chaque requête**, sur le cache, jamais en masquant dans l'interface.
- Une seule fonction d'autorisation (`can(user, action, resource)`) appelée partout, et une **suite de tests d'isolation** qui vérifie qu'un utilisateur ne voit jamais un logiciel non attribué (liste, recherche, roadmap, pièces jointes, notifications).
- Les **pièces jointes** et les **notifications e-mail** suivent les mêmes droits : un e-mail n'envoie jamais le contenu d'une demande à quelqu'un qui ne peut plus la voir.
- Toute modification de rôle, de groupe ou d'attribution est **auditée** (qui, quoi, quand).

### Côté développeurs

Les développeurs n'ont pas besoin de compte portail : leurs droits sont ceux de GitHub. S'ils veulent utiliser la vitrine, ils se connectent comme les autres et reçoivent un rôle. Une synchronisation avec les équipes GitHub (« l'équipe GitHub X a `contributor` sur ce logiciel ») est possible en V2, pour ceux qui ont un compte.

## 5 ter. Expérience : une application qui ne fait pas « tech »

Deux mondes, deux densités : l'espace équipe est calme et guidé ; l'espace triage (admins, triagers) peut être dense. GitHub ne transparaît jamais dans le premier.

### Le vocabulaire

| Jamais | À la place |
|---|---|
| issue, ticket | **demande** |
| repository, repo | **logiciel** (avec son nom et son icône) |
| open / closed, label, milestone | statut en mots simples, **version** |
| PR, merge, deploy | « en cours de réalisation », « en test », « **disponible** » |
| #1234 | le titre ; le numéro seulement en petit, pour en parler à un dev |
| assignee | « **pris en charge par** Enzo » (avec avatar) |

### Les écrans

- **Accueil** : « Bonjour Marie » ; ses demandes qui ont bougé depuis sa dernière visite ; « Nouveautés » des logiciels qu'elle utilise ; un gros bouton **« J'ai besoin de quelque chose »**.
- **Dépôt guidé, une question à la fois** : quel logiciel (cartes avec icônes) → « c'est un problème, une idée ou une question ? » → description libre → captures. Le formulaire GitHub du dépôt est rendu ainsi, champ par champ, avec une barre de progression. Brouillon conservé si on ferme l'onglet.
- **Captures sans effort** : coller depuis le presse-papiers, glisser-déposer, ou **« Capturer mon écran »** (API de capture du navigateur), avec annotation simple (flèche, rectangle, flou pour masquer une donnée).
- **Doublons à la saisie** : « Ça ressemble à une demande existante : *Export PDF illisible*. **C'est aussi mon cas** » (ajoute la personne comme concernée, au lieu d'un doublon).
- **Suivi façon colis** : une frise horizontale Reçue → Planifiée → En réalisation → En test → **Disponible**, l'étape en cours mise en valeur, une phrase humaine en dessous (« Prévue pour la version de novembre »). C'est le cœur de l'expérience.
- **Fil de la demande** : une conversation, comme une messagerie, pas une page de ticket. Les messages de l'équipe tech publiés vers le portail y apparaissent avec le nom et la photo de la personne.
- **« Ce qui arrive »** (roadmap) : trois colonnes **Maintenant / Bientôt / Plus tard**, des cartes avec un titre et une phrase, jamais de diagramme de Gantt ni de dates précises par défaut.
- **« Nouveautés »** : un fil de cartes « Disponible depuis le 12 novembre : l'export PDF est réparé », par logiciel.

### Le ton et la forme

- Phrases courtes, tutoiement ou vouvoiement configurable, aucun jargon, des états vides humains (« Rien en attente. Tout roule. »).
- Typographie généreuse, beaucoup d'air, couleur par logiciel pour se repérer, mode sombre, mobile d'abord pour le dépôt (on signale souvent un problème depuis son téléphone).
- **E-mails comme des messages** : « Ta demande *Export PDF illisible* est maintenant disponible », un bouton, rien d'autre.
- Accessibilité AA minimum, navigable au clavier.

### Direction visuelle retenue : « Franc » (2026-10-25)

- **Une couleur à plat par logiciel** (lavande, orange, menthe, jaune…), qui sert de repère partout : tuiles, liserés, en-tête de demande.
- **Contours noirs épais** (2,5 à 3 px), **ombres décalées franches** sur les éléments d'action, coins arrondis modérés.
- Typographie : **Bricolage Grotesque** pour les titres, **Atkinson Hyperlegible** pour le texte (conçue pour la lisibilité).
- Fond chaud cassé `#F4F1EC`, jaune `#FFE066` comme couleur d'action principale.
- Ton familier et direct (« Salut Marie, on fait quoi aujourd'hui ? », « Moi aussi ! »).
- À éviter : tout ce qui fait « SaaS généré par IA » (dégradés, bleu par défaut, cartes fades).
- Maquettes : https://claude.ai/artifact/AZxi4eSp1sKVLwXqpzb483

### Le titre public : écrit par GitHub, lisible par tous

Les titres tech (« fix(pdf): handle CMYK in renderer ») ne sont pas montrables. Un **champ d'issue d'organisation « Titre public »** (et éventuellement « Résumé public ») porte la version lisible : la source de vérité reste GitHub. Il est rempli par la personne qui trie ou, en option, **proposé par un modèle d'IA** à partir de l'issue et validé d'un clic. Sans titre public, le portail affiche le titre d'origine de la demande telle que déposée.

### L'IA, en option et discrète (V2)

- Reformuler une demande floue avant envoi (« Voici ce que j'ai compris : … C'est bien ça ? »), suggérer le logiciel et le type.
- Proposer les titres publics et les textes de « Nouveautés » à partir des releases.
- Désactivable, avec la clé de l'organisation (aucune clé fournie par le projet) ; aucune donnée envoyée sans réglage explicite.

## 5 quater. Création guidée des fonctionnalités et des issues

Objectif (Enzo, 2026-10-25) : **une fonctionnalité ou une issue ne se crée qu'en étant guidée champ par champ**, pour qu'elle soit à la fois **compréhensible par un humain** et **exécutable par un agent** (Claude) sans aller-retour.

### Principes

- **Un assistant pas à pas**, une étape par écran, avec un exemple sous chaque champ et un bouton « Proposer » (l'IA remplit à partir de ce qui est déjà saisi, l'humain valide).
- **Les champs viennent du dépôt** : ils sont définis dans les formulaires d'issue GitHub (`.github/ISSUE_TEMPLATE/feature.yml`, `task.yml`, `bug.yml`). Le portail les lit et les déroule en étapes. Modifier un champ = modifier le formulaire dans le dépôt.
- **Corps Markdown à titres fixes**, toujours dans le même ordre, précédé d'un marqueur de version (`<!-- hublot:issue v1 -->`). Un humain le lit comme un document ; un agent le découpe sans ambiguïté. Le marqueur permet de faire évoluer le gabarit sans casser les anciennes issues.
- **Les métadonnées vont dans les champs GitHub** (type d'issue, champs d'organisation Priorité, Effort, Date visée), jamais dans le texte.
- **Une jauge « Prête »** visible tout au long de la saisie, et sur chaque issue dans les listes.

### Champs d'une fonctionnalité

| Champ | Obligatoire | Va dans |
|---|---|---|
| Logiciel | oui | dépôt cible |
| Titre interne | oui | titre de l'issue Feature |
| Titre public | oui | champ d'issue « Titre public » |
| Le problème (qui est gêné, situation actuelle) | oui | corps · `## Problème` |
| Ce qui change pour l'utilisateur | oui | corps · `## Résultat attendu` (et texte public) |
| Utilisateurs concernés | oui | corps · `## Utilisateurs concernés` |
| Critères de réussite (mesurables) | oui, au moins 1 | corps · `## Critères de réussite` |
| Hors périmètre | conseillé | corps · `## Hors périmètre` |
| Version, date visée, priorité | oui | milestone, champs Date visée et Priorité |
| Demandes à l'origine | si elles existent | liens dans le corps |
| Découpage en issues | oui, au moins 1 | sous-issues |

### Champs d'une issue (tâche ou bug)

| Champ | Obligatoire | Va dans |
|---|---|---|
| Type (Tâche, Bug) et fonctionnalité parente | oui | type d'issue, sous-issue de |
| Titre (verbe à l'infinitif : « Répéter l'en-tête… ») | oui | titre |
| Contexte : pourquoi cette issue | oui | `## Contexte` |
| Ce qu'il faut faire | oui | `## À faire` |
| Bug seulement : étapes pour reproduire, obtenu, attendu, environnement | oui pour un bug | `## Reproduction` |
| Critères d'acceptation, vérifiables (« Étant donné… quand… alors… ») | oui, au moins 2 | `## Critères d'acceptation` (cases à cocher) |
| Zones concernées (modules, fichiers, écrans) | oui | `## Zones concernées` |
| Hors périmètre | conseillé | `## Hors périmètre` |
| Dépendances | si elles existent | dépendances GitHub « bloquée par » |
| Comment vérifier (tests, commande, parcours manuel) | oui | `## Vérification` |
| Priorité, effort (S, M, L), assignée à (une personne ou Claude) | oui | champs d'issue, assignation |

### Définition du « prêt »

Une issue est **prête** quand tous les champs obligatoires sont remplis, qu'elle a au moins deux critères d'acceptation vérifiables, au moins une zone concernée et une façon de vérifier. Le portail calcule la jauge ; une issue prête reçoit le label `pret` (et `pret-agent` si elle est assignée à Claude). Une issue qui n'est pas prête ne peut pas passer en « En réalisation » depuis le portail.

## 5 quinquies. Serveur MCP : créer depuis un prompt Claude

Demandé par Enzo (2026-10-25) : depuis Claude (Claude Code, l'app), un prompt du type « crée la fonctionnalité export Excel pour NovaMoon Ads, découpée en issues » doit aboutir à **une fonctionnalité et toutes ses issues**, avec les mêmes champs que la création guidée.

### Principes

- **Serveur MCP intégré au portail**, transport HTTP « streamable » sur `/mcp`. Pas de processus à installer côté utilisateur.
- **Claude agit au nom d'une personne** : authentification OAuth (flux standard MCP) ou jeton personnel. Le **RBAC s'applique à l'identique** : Claude ne voit et ne crée que ce que la personne peut voir et créer.
- **Mêmes règles que l'interface** : les schémas d'entrée des outils reprennent les champs des formulaires GitHub ; la définition du « prêt » est vérifiée côté serveur. Un appel incomplet n'échoue pas en silence : il répond avec la liste des champs manquants, pour que Claude pose les questions.
- **Brouillon par défaut**, réglable : ce que Claude crée arrive en brouillon (visible dans le portail, pas encore dans GitHub) jusqu'à validation d'un humain, ou directement dans GitHub si l'admin l'autorise.
- **Une seule opération pour tout l'arbre** : `create_feature` accepte la fonctionnalité et ses issues, et crée l'issue Feature puis les sous-issues et leurs dépendances de façon idempotente (clé d'idempotence fournie par l'appel), pour qu'une relance ne duplique rien.
- **Tout est journalisé** avec la mention « via Claude » et la personne pour qui Claude agissait.

### Outils exposés (v1)

| Outil | Fait |
|---|---|
| `list_software` | Les logiciels accessibles, avec leurs formulaires et champs |
| `search` | Cherche demandes, fonctionnalités et issues (texte, logiciel, statut) |
| `get_feature`, `get_issue` | Le détail, avec la jauge « prête » et les champs manquants |
| `create_feature` | Une fonctionnalité **avec ses issues** en un appel (brouillon ou GitHub) |
| `create_issue`, `update_issue` | Une issue seule, rattachée ou non à une fonctionnalité |
| `check_ready` | Valide un brouillon sans rien créer : ce qui manque, champ par champ |
| `link_requests` | Rattache des demandes de l'équipe à une fonctionnalité (les personnes seront prévenues) |
| `triage` | Classe une demande (type, priorité, logiciel), selon les droits |
| `roadmap` | Ce qui arrive, par logiciel |

Ressources MCP en lecture : le gabarit Markdown versionné et la définition du « prêt », pour que Claude rédige au bon format sans deviner.

### Écran de configuration (admin › Réglages › Claude et MCP)

- Activer le serveur, voir son adresse, copier la commande d'ajout (`claude mcp add --transport http hublot <url>/mcp`).
- Mode de création : brouillon à valider, ou directement dans GitHub.
- Outils activés, un par un.
- Jetons personnels : créer, nommer, voir la dernière utilisation, révoquer.
- Journal des dernières actions faites via Claude.

## 6. Correspondance avec GitHub

| Notion du portail | Élément GitHub | API |
|---|---|---|
| Organisation | Organisation | GitHub App installée sur l'organisation |
| Logiciel | Dépôt (liste configurée) | Repositories |
| Demande | Issue créée par l'app, label `portal` | Issues REST |
| Type (bug, idée, feature, tâche) | Type d'issue d'organisation | `/orgs/{org}/issue-types` |
| Priorité, effort, dates | **Champs d'issue d'organisation** (GA juillet 2026) | Issue field values, REST et GraphQL |
| Statut détaillé | Champ Statut d'un Project, à défaut ouvert/fermé + raison | Projects v2 REST/GraphQL |
| Initiative → feature → tâche | Sous-issues (100 par parent, inter-dépôts) | Sub-issues REST |
| Bloqué par | Dépendances d'issue (50 par type) | Issue dependencies REST |
| Version | Milestone, Release | Milestones, Releases |
| En cours / en revue | PR liée | GraphQL `closingIssuesReferences`, timeline |
| Livré en test / en prod | Déploiement réussi sur un environnement | Deployments, Environments |
| Envoi vers le bon logiciel | Transfert d'issue | GraphQL `transferIssue` |
| Formulaire de dépôt | Issue forms YAML | Repository contents |
| Vote | Réaction 👍 | Reactions |
| Mise à jour en temps réel | Webhooks `issues`, `issue_comment`, `sub_issues`, `projects_v2_item`, `deployment_status`, `release`, champs d'issue | Webhooks |

**Identité de l'auteur.** Les issues sont créées par la GitHub App. Le corps contient l'auteur réel (nom, e-mail masqué ou non selon la configuration) et un identifiant de demande portail. Le portail garde le lien personne ↔ issue.

## 7. Données stockées par le portail

| Table | Contenu |
|---|---|
| `user` | e-mail, nom, rôle d'organisation, langue, préférences de notification, statut (actif, suspendu) |
| `group`, `group_member` | groupes et leurs membres |
| `project` | logiciel : dépôt GitHub, nom affiché, réglage « les reporters voient tout » |
| `grant` | attribution d'un rôle par logiciel à une personne **ou** à un groupe |
| `audit_log` | changements de rôles, de groupes, d'attributions, actions de triage |
| `login_token` | jeton de lien magique (haché), expiration |
| `session` | sessions actives |
| `request` | lien personne ↔ issue (repo, numéro, node id), date de dépôt |
| `attachment` | clé objet, type, taille, issue liée, auteur |
| `issue_cache` | projection des issues, champs et statuts, alimentée par webhooks |
| `config` | logiciels, correspondance des statuts, règles de visibilité (aussi lisible depuis un fichier) |
| `webhook_delivery` | id de livraison GitHub, pour l'idempotence |

Rien d'autre. Le cache est jetable : il se reconstruit depuis GitHub.

## 8. Architecture

```
 Utilisateurs non tech ──► SPA (React) ──► API du portail ──► GitHub API (App)
                                             │   ▲
                                 Postgres ◄──┤   └── Webhooks GitHub
                                 S3 ◄────────┘        (file de jobs)
                                 SMTP (liens magiques, notifications)
```

- **GitHub App** (pas de jeton personnel) : droits minimaux. Issues lecture/écriture, Metadata lecture, Contents lecture (formulaires), Projects lecture/écriture, Deployments lecture, Organisation : issue types et issue fields lecture.
- **Webhooks → file de jobs → cache.** Chaque livraison est vérifiée (signature HMAC), dédoublonnée par son id, puis traitée en arrière-plan. Une resynchronisation complète périodique répare ce qu'un webhook perdu aurait raté.
- **Écritures** : directement vers GitHub, puis cache mis à jour à la réception du webhook (ou optimiste).
- **Adaptateur de forge** : tout l'accès GitHub passe par une interface métier (`CreateRequest`, `ListRoadmap`…), ce qui garde la porte ouverte à GitLab plus tard sans le promettre.

## 9. Stack proposée

**Recommandation : Go + React, un seul binaire.**

| Couche | Choix | Pourquoi |
|---|---|---|
| Backend | **Go**, `chi`, `pgx` + `sqlc`, `goose` | Ce que tu maîtrises déjà (media-poste-ads). Binaire statique, image de quelques dizaines de Mo. |
| GitHub | `google/go-github` (REST), `shurcooL/githubv4` (GraphQL), `bradleyfalzon/ghinstallation` (auth GitHub App) | Bibliothèques mûres. Les nouveautés récentes (champs d'issue, dépendances) se complètent par appels REST directs si go-github est en retard. |
| Jobs | **River** (sur Postgres) | Webhooks, resynchronisation, notifications, sans Redis. |
| Base | **PostgreSQL** | Une seule dépendance d'état. |
| Fichiers | Stockage compatible **S3** (OVH Object Storage, MinIO, R2) | Les pièces jointes que GitHub ne prend pas par API. |
| Frontend | **React + Vite + TypeScript**, Tailwind, TanStack Query, `react-i18next` | Même stack que media-poste-ads ; SPA servie par le binaire Go (`embed`). |
| Auth | Lien magique maison + OIDC optionnel (`coreos/go-oidc`) | Pas de mot de passe stocké. |
| E-mail | SMTP standard | Fonctionne partout, y compris auto-hébergé. |
| Déploiement | **Une image Docker** + Postgres + S3 ; `compose.yaml` fourni ; Helm plus tard | « `docker compose up` et ça marche » est l'argument numéro un d'un projet open source. |

**L'alternative écartée : Next.js / TypeScript de bout en bout.** Avantages réels : Octokit est le SDK officiel de GitHub (auth d'app, webhooks, pagination, gestion des quotas intégrés) et le vivier de contributeurs TypeScript est plus large. Écartée parce que tu iras plus vite et plus sûrement en Go, que l'auto-hébergement d'un binaire unique est plus simple qu'un serveur Node, et que la stack serait la même que media-poste-ads (patterns, outillage, revues réutilisables). À reconsidérer si l'objectif devient avant tout de maximiser les contributions externes.

## 10. Sécurité et confidentialité

- **Le portail a plus de droits que ses utilisateurs** : chaque lecture est filtrée côté serveur (logiciels autorisés, issues étiquetées `portal` ou visibles par règle, commentaires internes exclus). Jamais de passe-plat vers l'API GitHub.
- Liens magiques à usage unique, courts (15 min), hachés en base, limités en fréquence.
- Pièces jointes : vérification du type réel, taille maximale, URL signées à durée courte ; aucune pièce jointe publique par défaut.
- Webhooks : signature vérifiée, idempotence par id de livraison.
- Clé privée de la GitHub App hors du dépôt (fichier monté ou coffre).
- Journal d'audit des actions admin (triage, transfert, configuration).
- RGPD : données personnelles minimales, suppression d'un utilisateur possible (ses issues restent dans GitHub, anonymisées dans le corps si demandé).

## 11. Paysage existant (recherche du 2026-10-25)

| Outil | Ce qu'il fait | Pourquoi il ne suffit pas |
|---|---|---|
| **Quackback** (AGPL-3.0, auto-hébergeable) | Retours, votes, roadmap, changelog, helpdesk ; droits par tableau dont anonymes ; lien GitHub qui fait suivre le statut | Ses propres posts sont la référence ; une demande est « promue » en issue à la main ; la roadmap n'est pas celle de GitHub |
| Canny, Featurebase (SaaS) | Portail d'idées avec synchro GitHub | Payants, centrés sur les retours clients, pas sur la vue du travail |
| Marker.io, Jam, BugHerd | Signalement de bug avec capture, sans compte | Pas de suivi pour le signaleur, pas de roadmap |
| ZenHub | Epics, sprints, tableaux multi-dépôts | Tout le monde doit avoir un compte GitHub |
| FasterFixes, feedbacks-oss | Widget qui crée des issues | Pas de retour de statut confirmé |
| Astuto | Retours auto-hébergés | Archivé en février 2026 |

**Positionnement** : le seul outil qui soit **une vue de GitHub lui-même**, multi-dépôts, pour des personnes sans compte, avec le cycle de vie jusqu'à la mise en production.

Décision (2026-10-25) : **aucun de ces outils n'est testé ni étudié en détail**, pour concevoir le portail sans biais. Ce tableau sert seulement à situer le projet.

## 12. Open source

- **Porteur** : organisation GitHub d'ET Technology Consulting.
- **Licence : AGPL-3.0** (décidé le 2026-10-25). Quiconque exploite le portail en service doit publier ses modifications, ce qui protège une éventuelle offre hébergée d'ET Technology Consulting.
  - **Un CLA dès la première contribution externe**, qui cède ou concède les droits à ET Technology Consulting. Sans lui, impossible plus tard de proposer une licence commerciale (double licence) ou de changer de licence. Un simple DCO ne suffit pas pour ça.
  - Droits d'auteur au nom d'ET Technology Consulting, en-tête SPDX `AGPL-3.0-or-later` dans chaque fichier source.
  - Dépendances : vérifier qu'elles sont compatibles (MIT, BSD, Apache-2.0, MPL le sont ; c'est le cas des bibliothèques de la stack proposée).
  - La SPA servie au navigateur est couverte aussi : le code source doit être accessible depuis l'interface (lien « Code source » dans le pied de page, exigé par l'article 13 de l'AGPL).
- **Publication** après quelques semaines d'usage interne chez NovaMoon, avec : README, guide d'installation en 5 minutes, création guidée de la GitHub App (manifeste d'app GitHub), `SECURITY.md`, `CONTRIBUTING.md`, démo en ligne.
- **Développement indépendant** : aucun code ni design repris d'un outil existant, ce qui garde aussi la propriété intellectuelle nette pour le CLA et une éventuelle double licence.

## 13. Limites connues de GitHub

- **Pas d'API pour joindre une image** à une issue ou un commentaire : d'où le stockage S3 du portail.
- **Quotas** : environ 5 000 requêtes/heure par installation, budget en points pour GraphQL ; le cache alimenté par webhooks est obligatoire.
- **Webhooks** : URL publique requise ; en développement, tunnel ou smee.io.
- **Création des champs et types d'issue** réservée aux propriétaires de l'organisation : le portail les lit, l'admin les définit dans GitHub.
- Champs d'issue et dépendances sont récents (2025-2026) : surveiller le changelog GitHub, les API peuvent encore bouger.

## 14. Risques

| Risque | Parade |
|---|---|
| Glisser vers un deuxième Linear | Principe 4, revu à chaque fonctionnalité |
| GitHub sort la même chose nativement | Peu probable pour l'accès sans compte ; le portail reste une couche fine, facile à faire évoluer |
| Quotas sur une grosse organisation | Cache, webhooks, GraphQL groupé, resynchronisation incrémentale |
| Fuite d'informations internes vers des non-membres | Filtrage serveur strict, tests dédiés, règles de visibilité explicites |
| Maintenance open source | Publier tard, périmètre étroit, issues de contribution bien étiquetées |

## 15. Indicateurs de réussite

- Une personne non tech dépose une demande complète en **moins de 2 minutes**.
- **Zéro** demande perdue entre le portail et GitHub.
- Les demandes de l'équipe passent par le portail plutôt que par message direct (objectif : 80 % après un mois).
- Au moins un outil payant abandonné chez NovaMoon.
- Installation d'une instance neuve en **moins de 15 minutes** par un tiers.

## 16. Jalons

| Jalon | Contenu | Estimation |
|---|---|---|
| J0 | Maquette des écrans clés, création de la GitHub App, mise en place des types, champs et labels sur NovaMoonMedia | 1 jour |
| J1 | Squelette Go + React, lien magique, config des logiciels, lecture des formulaires, dépôt d'une demande avec pièce jointe | 3 à 4 jours |
| J2 | Webhooks, cache, « Mes demandes », triage, notifications e-mail | 3 jours |
| J3 | Usage interne NovaMoon, corrections | 2 à 3 semaines d'usage |
| J4 | V2 vitrine : roadmap, cycle de vie jusqu'à la prod, releases | 1 à 2 semaines |
| J5 | Publication open source | 2 à 3 jours de préparation |

## 17. Questions ouvertes

1. Nom du projet.
2. ~~Licence~~ : AGPL-3.0, décidé le 2026-10-25. Reste à choisir le texte du CLA.
3. Un dépôt `retours` commun pour tout ce qui n'est pas encore trié, ou dépôt direct dans le logiciel choisi ?
4. ~~Visibilité des non-tech~~ : par logiciel, via le RBAC (§5 bis). Le réglage « les reporters voient toutes les demandes » est **désactivé par défaut** (décidé le 2026-10-25) : on ouvre plus facilement qu'on ne referme.
5. Quels commentaires GitHub deviennent visibles dans le portail : préfixe, label, ou réponse explicite depuis le portail uniquement ?
6. Hébergement de l'instance NovaMoon : OVH (cohérent avec media-poste-ads) ?
7. Offre hébergée payante un jour, oui ou non ? (oriente la licence)

## Sources

- [Dépendances d'issue, API REST](https://docs.github.com/en/rest/issues/issue-dependencies) ; [annonce](https://github.blog/changelog/2025-08-21-dependencies-on-issues/)
- [Sous-issues en REST](https://github.blog/changelog/2024-12-12-github-issues-projects-close-issue-as-a-duplicate-rest-api-for-sub-issues-and-more/) ; [`gh` et sous-issues, types, dépendances](https://github.blog/changelog/2026-06-10-manage-sub-issues-types-and-dependencies-from-github-cli/)
- [Champs d'issue disponibles pour tous](https://github.blog/changelog/2026-07-02-issue-fields-are-now-generally-available/) ; [API des valeurs de champ](https://docs.github.com/en/rest/issues/issue-field-values) ; [gestion des champs](https://docs.github.com/en/issues/tracking-your-work-with-issues/using-issues/managing-issue-fields-in-your-organization)
- [Recherche avancée d'issues dans l'API](https://github.blog/changelog/2025-03-06-github-issues-projects-api-support-for-issues-advanced-search-and-more/)
- [Projects via l'API](https://docs.github.com/en/issues/planning-and-tracking-with-projects/automating-your-project/using-the-api-to-manage-projects) ; [éléments REST](https://docs.github.com/en/rest/projects/items) ; [champs REST](https://docs.github.com/en/rest/projects/fields) ; [événements webhook](https://docs.github.com/en/webhooks/webhook-events-and-payloads)
- [Absence d'API pour les pièces jointes](https://braindetox.kr/en/posts/gh_attach_cli_2026.html) (article communautaire)
- [Quackback](https://github.com/QuackbackIO/quackback) ; [synchro avec les trackers](https://quackback.io/blog/sync-feedback-to-issue-tracker)
