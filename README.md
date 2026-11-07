# Borda

Console local para a borda de uma rede. Mostra por onde a casa está saindo, se a saída reserva por USB está pronta, e quem está na LAN. O RouterOS continua dono do resto.

A fatia 1 está no repositório: uma API em Go e um frontend SvelteKit, em português e inglês. Ela só lê. A fatia 2 prova a saída USB: sem confirmação, nada muda; com confirmação, só a rota comentada `borda-primary` pode ser desabilitada, e ela é reabilitada ao final. Um CHR 7.25.1 de laboratório, em 127.0.0.1, mostrou a rota padrão ativa e a checagem respondendo. A prova confirmada não começou, porque esse CHR não tem LTE nem o comentário `borda-primary`, e a rota do DHCP continuou ativa. A medida pela USB ainda não foi feita num roteador de verdade. A fatia 3 abre um aparelho da leitura atual: endereço, nome anunciado e o caminho ativo do roteador, sem gravar firewall nem rota. No CHR, isso foi o gateway visto por ARP. Visto há pouco e ausente ficam de fora enquanto não houver tempo de guarda. A fatia 4 guarda a troca de caminho no hub, com hora e motivo. O roteador não é o arquivo. No CHR, com o cofre fechado, a tela de trocas abriu vazia e a rota padrão continuou ativa. Os motivos da lista fechada ficaram no teste: esse CHR não trocou de caminho e a prova não começa. Nada é apagado enquanto o tempo de guarda não estiver decidido. A fatia 5 responde pelo retrato filtrado e a única proposta é a prova da USB. Fora da lista, nada é executado. Sem confirmação, a prova não começa. O modelo ainda não foi escolhido. No CHR, com o cofre fechado, um pedido de export foi recusado e a rota padrão continuou ativa. A fatia 6 aprova o agent: o hub recebe o retrato e os eventos, e a senha fica no agent. No mesmo CHR, com o cofre fechado, a aprovação mostrou a sessão fechada e nenhuma troca, sem pedir senha. Um comando de escrita enviado ao hub não roda. O endereço do hub remoto ainda não foi escolhido. Não aponta para um roteador de produção. O plano está em [docs/plano-de-desenvolvimento.md](docs/plano-de-desenvolvimento.md).

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
