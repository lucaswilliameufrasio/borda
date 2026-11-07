# Borda

Console local para a borda de uma rede. Mostra por onde a casa está saindo, se a saída reserva por USB está pronta, e quem está na LAN. O RouterOS continua dono do resto.

A fatia 1 está no repositório: uma API em Go e um frontend SvelteKit, em português e inglês. Ela só lê. Não aponta para um roteador de produção. A jornada com RouterOS de laboratório ainda não foi executada. O plano está em [docs/plano-de-desenvolvimento.md](docs/plano-de-desenvolvimento.md).

## API

```sh
go test ./...
go run ./cmd/borda -listen 127.0.0.1:8787 -data ./data -reader memory
```

O modo `memory` não abre SSH. Ele responde por quatro endereços de exemplo: `connected.example`, `other.example`, `empty.example` e `offline.example`. Qualquer senha não vazia serve para esses exemplos. O modo `ssh`, que é o padrão, usa o host que a pessoa cadastrou e guarda a senha só na memória do processo.

A API escuta apenas em loopback. O cadastro fica em `./data`, que não entra no Git.

## Frontend

```sh
cd web
npm install
npm test
npm run dev
```

O dev server usa `http://127.0.0.1:5173` e fala com `http://127.0.0.1:8787`. Para outro endereço, defina `PUBLIC_BORDA_API_URL`.
