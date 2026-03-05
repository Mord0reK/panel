# panel-integrations

Standard tworzenia integracji usług zewnętrznych w backendzie panelu.

## Cel
- Każda integracja ma własny plik Go w `backend/internal/integrations/`.
- API usług działa pod prefiksem `/api/services/{service}/...`.
- Konfiguracja (enabled, URL, auth) jest zarządzana wyłącznie przez panel.
- Sekrety są szyfrowane kluczem pochodnym z `JWT_SECRET`.

## Struktura integracji
1. Dodaj plik usługi, np. `backend/internal/integrations/adguardhome.go`.
2. Zaimplementuj metadane `Definition()`:
   - `key` (np. `adguardhome`),
   - `display_name`,
   - `requires_base_url`,
   - `auth_type` (`token` lub `basic_auth`),
   - listę endpointów (`/services/{service}/...`).
3. Dodaj usługę do `registered` w `registry.go`.

## Konfiguracja i baza
- Dane usługi trzymamy w `service_integrations`:
  - `service_key`,
  - `enabled`,
  - `base_url`,
  - `encrypted_token`,
  - `encrypted_username`,
  - `encrypted_password`.
- Nigdy nie zapisuj sekretów w plaintext.
- API nie zwraca sekretów — tylko flagi obecności.

## Endpointy
- Globalna lista usług:
  - `GET /api/services`
- Konfiguracja usługi:
  - `GET /api/services/{service}/config`
  - `PUT /api/services/{service}/config`
- Test połączenia:
  - `POST /api/services/{service}/test`
- Dane usługi:
  - `GET /api/services/{service}/{custom}`

## Frontend (Ustawienia > Integracje)
- Render formularza oparty o metadane z backendu:
  - `requires_base_url=true` => pokaż pole URL,
  - `auth_type=token` => pokaż token,
  - `auth_type=basic_auth` => pokaż username/password.

## Bezpieczeństwo
- Waliduj `base_url` i chroń przed SSRF.
- Używaj timeoutów i bezpiecznych retry.
- Nigdy nie loguj sekretów.
- Traktuj brak `JWT_SECRET` jako błąd krytyczny konfiguracji.

## Checklista DoD dla nowej integracji
- [ ] Metadane dodane w osobnym pliku integracji.
- [ ] Usługa dodana do rejestru.
- [ ] Endpoint(y) `/api/services/{service}/...` dodane.
- [ ] Konfiguracja i sekrety obsłużone (szyfrowanie/dekryptacja).
- [ ] Testy unit + integration przechodzą.
- [ ] Dokumentacja endpointów zaktualizowana.
