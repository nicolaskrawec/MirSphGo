# MirSphGo - Optimized Scalable Ray Tracer

MirSphGo est un moteur de rendu par lancer de rayons (Ray Tracer) écrit en Go utilisant la bibliothèque **Ebitengine**. 

Ce projet a été réalisé de manière assez intensive et, pour tout dire, c'est **pas mal vibecodé** !

## Caractéristiques

- **Multi-threading** : Utilise tous les cœurs de votre CPU pour accélérer le rendu.
- **Réflexions** : Gestion des matériaux réfléchissants avec récursion ajustable.
- **Éclairage dynamique** : Ombres portées et modèle d'éclairage Blinn-Phong.
- **Caméra interactive** :
    - Déplacement avec **Z/S/Q/D** (ou W/A/S/D).
    - Rotation à la **souris**.
    - Fenêtre **redimensionnable**.
- **Animations** : Système de sphères orbitales animées en temps réel.
- **Architecture modulaire** : Code découpé en plusieurs fichiers Go (`vec3`, `ray`, `material`, `shape`, `scene`, `game`) pour une meilleure clarté.

## Installation et Utilisation

### Prérequis

- [Go](https://golang.org/dl/) installé sur votre machine.

### Lancement

Pour lancer le projet en mode développement :

```bash
go run .
```

### Commandes

- **Z / W** : Avancer
- **S** : Reculer
- **Q / A** : Gauche
- **D** : Droite
- **Souris** : Tourner la vue
- **Echap** : Quitter

## Compilation Optimisée

Pour générer un exécutable optimisé et léger (Windows) :

```bash
go build -ldflags="-s -w" -o MirSphGo.exe .
```

---
*Projet développé avec passion et un bon paquet de vibes.*
