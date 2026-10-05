---
review_status: APPROVED_WITH_OBSERVATIONS
scope: release.sh, release_test.go
---

# Revision del flujo de release

Veredicto: APPROVED_WITH_OBSERVATIONS. Sin bloqueantes en el cambio revisado.

## Hallazgos

### P2: el test de build puede pasar por un fallo posterior

Referencias: release_test.go:62, release_test.go:102, release.sh:85.
El mock falla todas las compilaciones y no deja el binario del host. El test
acepta cualquier salida no cero y ausencia de publicacion, sin comprobar que
no se ejecutaron mas builds, tag, push o create despues del fallo de build.
Una copia temporal con `|| true` agregado a go build sigue pasando el caso
build-fails: cp falla posteriormente por ausencia del binario local.
Esto es una laguna de cobertura del test nuevo, no un bug demostrado en el
script actual, cuyo set -e si detiene la compilacion fallida.
Recomendacion: comprobar que se alcanzo el comando fallido y que no se
ejecutaron etapas posteriores; cubrir fallo de un build posterior y de push.

## Comportamiento revisado

- release.sh:15 y :21: gh y autenticacion se exigen antes de invocar Git.
- release.sh:67 y :88: build y tag no tienen supresion de errores.
- release.sh:91 y :92: los push se ejecutan directamente bajo set -e.
- release.sh:95: lista explicita de cuatro binaries; excluye assets obsoletos.
- release.sh:96: DMG opcional, ruta citada mediante expansion del array.
- release.sh:99: create usa --verify-tag --draft y repo explicito.
- release.sh:100: edit --draft=false --latest solo corre si create termina bien.
- release.sh:102: anuncio de exito solo despues de editar/publicar sin error.
- No se detectaron nuevas interpolaciones inseguras, eval ni exposicion de
  credenciales. El estado de autenticacion se consulta sin imprimir secretos.
- Las construcciones nuevas son compatibles con Bash 3.2; ejecucion validada
  en Bash 5.2.21. No se ejecuto Bash 3.2 ni una llamada gh real.

## Verificacion

- bash -n release.sh: correcto.
- go test -count=1 -v .: ocho casos pasan, shell real con Git/Go/gh simulados.
- go test -count=1 ./...: todas las pruebas pasan.
- go vet ./...: correcto.
- git diff --check y gofmt -l release_test.go: sin incidencias.
- Mutacion temporal fuera del repositorio: build-fails sigue pasando al
  ocultar los errores de go build; confirma el hallazgo P2.

## Riesgos preexistentes

El tag se sube antes de publicar. Un fallo de create/edit deja tag remoto sin
Release publico; upgrade descubre tags y puede intentar descargar assets 404.
El cambio impide anunciar exito pero no hace atomico ese flujo ni repara los
tags v2.3.6/v2.3.7. El DMG existente puede estar desactualizado; su empaquetado
no forma parte del cambio. Auto git add/commit y codesign con || true ya
existian y no se consideran defectos introducidos ni bloqueantes de este MR.

## Limites

No se modifico codigo ni se crearon commits. Solo se escribe este informe.
La publicacion real y los assets remotos no se verificaron sin credenciales gh.

## Seguimiento de implementacion

Se atendio el hallazgo P2: los tests comprueban el ultimo comando ejecutado y
el numero de builds antes del fallo. Se agregaron fallos de un build posterior
y de ambos push. Los once casos, la suite completa, go vet, go build y la
sintaxis Bash pasaron. Se reinstalo y verifico v2.3.7 localmente.

El Release v2.3.7 sigue devolviendo HTTP 404. Instalen GitHub CLI desde
https://cli.github.com/ y autentiquen directamente en su terminal. Para reparar
el tag existente, sin incrementar la version:

```bash
gh auth login
gh release create v2.3.7 bin/cogni_{darwin,linux}_{arm64,amd64} \
  --repo AdelysAlberto/cogni-memory --verify-tag --draft \
  --title v2.3.7 --notes "Release v2.3.7"
gh release edit v2.3.7 --repo AdelysAlberto/cogni-memory --draft=false --latest
```

No se crearon nuevos tags ni se publico un Release durante esta correccion.

## Uso habitual

La autenticacion de GitHub CLI ya esta configurada. Para versiones nuevas,
todo el flujo de compilacion, tag, push, assets y publicacion se ejecuta con:

```bash
./release.sh patch
./release.sh minor
./release.sh major
```

Los comandos manuales de recuperacion anteriores solo reparan tags incompletos;
no forman parte de las publicaciones normales. Los trece escenarios pasan,
incluidos los incrementos patch, minor y major, las cuatro versiones embebidas
y el orden tag, push, borrador con assets y publicacion. go vet, la suite
completa, go build y git diff --check tambien pasaron. No se publico una nueva
version durante estas pruebas.
