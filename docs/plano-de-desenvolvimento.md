# Plano de desenvolvimento

Console local para a borda da rede. A pessoa abre e vê por onde a casa está saindo. O RouterOS continua sendo o lugar de firewall avançado, fila, VPN, usuário e script solto. Este repositório guarda o plano. Nenhum código de produto existe ainda.

## Resultado

A interface é um frontend separado. Ela fala com uma API em Go. Cada roteador tem um agent na LAN, e a API pode acompanhar vários agents ao mesmo tempo. O agent guarda a sessão daquele roteador e a credencial, só na memória. A interface mostra o caminho ativo, a prontidão da saída reserva e os aparelhos conhecidos de cada um. Toda escrita é uma ação nomeada, mostrada antes, e só segue depois de confirmação. Um modelo de linguagem pode explicar o estado ou propor uma ação dessa lista. Ele não recebe credencial, export, backup nem senha, e não grava configuração sozinho.

## Fora deste produto

- Reescrever o Winbox ou o WebFig. A tela de caminho pode ter um atalho para abrir o WebFig.
- Proxy livre de comando RouterOS. Texto livre não vira comando.
- Fila, VPN, usuário, firewall avançado e script solto.
- Banco de métricas dentro do roteador. O histórico fica no hub.
- Conta remota no modo local.

## Atores

| Ator | Pode | Não pode |
| --- | --- | --- |
| Morador | Ver se há saída, qual WAN está ativa e quais aparelhos estão na LAN. Pedir a prova da saída reserva e confirmar uma mudança de failover. | Gravar política sem a confirmação daquela ação. |
| Operador | O mesmo morador quando muda política: cadastrar uma WAN, mudar distância, desligar uma checagem. Confirma cada escrita. | Executar comando fora da lista fechada. |
| Agent | Único processo que guarda a sessão do roteador e a credencial, só na memória. | Enviar a senha ao hub, ao modelo ou ao log. |
| Hub | API, histórico e assistente. Acompanha um ou mais agents, um roteador por agent. | Guardar a senha do roteador. A interface não fala com o RouterOS. |
| Modelo | Ler um retrato já filtrado e devolver explicação ou proposta tipada. | Ter cliente do roteador ou cofre. |

Os dois modos de execução fazem parte do produto. No modo local, a API e os agents sobem nas máquinas da LAN e não há conta remota. No modo separado, cada agent fica na LAN do seu roteador e o hub pode estar em outro host. O hub guarda a identidade de cada agent e os retratos que eles enviam. Um comando do hub chega como pedido assinado. O agent só executa o que a política local permite, e uma escrita pede confirmação. A senha não atravessa esse canal.

## Jornadas

Cada jornada tem uma tela. Estado, cadastro e ação destrutiva não dividem o mesmo lugar.

### 1. Preparar a casa

Uma vez por roteador. O dia a dia não pede isso de novo. Incluir outro MikroTik repete esta jornada para aquele aparelho. A escolha fica nas configurações do tenant, fora da tela de caminho.

1. A pessoa informa o nome do tenant.
2. O hub lista os gateways das rotas desta máquina que aceitam SSH. Um só pede confirmação. Vários, a pessoa escolhe. Nenhum, ela digita o endereço.
3. A pessoa escolhe um cofre, vê o que vai digitar, e segue. A senha do roteador mora nesse cofre e não volta para a tela.
4. O hub lê a identidade do roteador e mostra a versão e o nome. A senha não entra nessa tela.
5. A pessoa cai no caminho ativo.

Cofres previstos: arquivo cifrado do próprio console, em que a pessoa cria a senha do cofre nessa tela; Ciphera; Infisical; chaveiro do sistema. Em todos os externos, o console lê o segredo na hora de usar e não copia o valor para o banco dele.

Vazio: nenhum gateway SSH. Falha: autenticação recusada ou roteador inalcançável, sem a senha na mensagem. Sucesso: identidade lida e caminho ativo aberto.

### 2. Ver o caminho ativo

A pessoa abre o console e vê qual rota padrão está ativa, se a checagem de alcance responde pela porta da WAN primária, e se a LAN conseguiria sair pela reserva USB. A reserva está pronta quando existem o NAT e o forward da LAN para a interface USB, nessa ordem, antes do descarte do resto.

Vazio: nenhuma rota padrão. Falha: agent sem sessão. Sucesso: um caminho nomeado e o tempo desde a última troca.

### 3. Provar a saída USB

Com o modem USB no cabo, dados móveis ligados e o Wi-Fi da casa desligado, a pessoa confirma a prova. O agent desabilita a rota primária, mede um destino público pela interface USB e pela LAN, e reabilita a rota. A casa fica sem a WAN primária por alguns segundos. Cancelar no meio reabilita a rota.

Sucesso: a medida volta e a rota primária está ativa de novo. Falha: a rota primária é reabilitada mesmo assim, e a tela diz que a prova não passou.

### 4. Cadastrar uma WAN

A pessoa escolhe uma porta que não é a porta da LAN, informa distância e nome, e confirma. O agent tira a porta da bridge, cria o NAT e o cliente DHCP nessa distância. Duas WANs não ficam com a mesma distância.

### 5. Ver aparelhos

A pessoa vê leases, vizinhos e o último anúncio. Abre um aparelho para ver endereço, nome e por qual caminho o tráfego dele saiu. Isso não altera firewall nem rota.

Estados do aparelho: visto agora, visto há pouco, ausente.

### 6. Perguntar ao assistente

A pessoa escreve o que está vendo. O modelo recebe só o retrato filtrado e pode propor uma ação da lista fechada. A proposta aparece para confirmar. Sem confirmação, nada é escrito. Se o retrato não bastar, a resposta diz o que falta ler. Ela não pede a senha.

O modelo vê nomes de interface, comentários, distâncias, flags, contagens e atrasos. Não vê senha, usuário, URL com credencial, conteúdo de usuário do roteador, export, backup, nem um script que carregue segredo.

### 7. Ligar um agent a um hub remoto

A pessoa aprova o agent na LAN. O hub passa a receber retrato e evento. A credencial continua no agent.

## Estados

- Caminho: cabo ativo, USB ativo, os dois inativos, prova em andamento.
- Política de failover: ordem das distâncias e checagem de cada WAN. A política desejada fica no hub. O RouterOS é a fonte do que está valendo agora.
- Sessão do agent: conectada, sem permissão de escrita, autenticada e recusada, roteador inalcançável.
- Troca de caminho é evento, com hora e motivo: a checagem falhou, a prova foi pedida, ou a rota foi reabilitada.

## Módulos

- Segredo. Um adaptador por cofre. O agent pede a senha na hora da sessão. Não há API para o resto do sistema ler o valor. Backup e export ficam no disco do agent, cifrados, e não entram no canal do hub nem no prompt.
- Sessão RouterOS. Operações nomeadas: ler rotas, ler forward, ler leases, desabilitar uma rota comentada, reabilitar, incluir uma WAN já desenhada. Timeout, erro legível, nenhuma senha na mensagem de erro.
- Política de caminho. Compara o desejado com o que o roteador tem. O endereço de checagem não pode ser gravado como DNS da LAN.
- Inventário. DHCP, ARP e neighbor, amarrados por MAC.
- Telemetria. Amostras de qual rota está ativa, perda e atraso da checagem, contadores das interfaces, presença de aparelho. A coleta para se o agent perder a sessão. Não inventa ponto.
- Assistente. Leitura sobre o retrato filtrado e uma lista fechada de propostas. A escrita passa pelo comando, com confirmação.
- Comando. Idempotência por comentário estável. Recusa comando fora da lista. Mostra o que vai mudar antes de aplicar.

Os comentários estáveis são o contrato das escritas. Os nomes públicos ainda não estão fechados. A leitura da fatia 1 não depende deles: ela mostra a rota padrão que o roteador já tem.

## Observabilidade

Separada do cadastro.

- O caminho: qual rota padrão está ativa, se a checagem responde pela WAN primária, se a interface USB tem endereço, se a sessão USB está de pé, e se o forward da LAN para o USB existe.
- A casa: leases ativos, aparelho novo, aparelho que sumiu, nome anunciado.
- O evento: troca de WAN, prova iniciada, prova concluída, agent sem sessão.

Um gráfico de vazão da interface pode entrar depois do evento de troca. Sem o evento, o gráfico não explica a queda. O gráfico de CPU e de porta que o RouterOS já publica não é esta tela.

## Fatias

Cada fatia fecha uma jornada e fica demonstrável sozinha. Nada abaixo está implementado.

### 1. Setup e caminho ativo, só leitura

A pessoa escolhe o cofre e o agent daquele roteador. A API em Go entrega o retrato. O frontend, separado, mostra a lista de roteadores e, ao abrir um, o caminho ativo, as rotas, o forward e os leases. A credencial fica fora do código, fora do log e fora do frontend.

| ID | Resultado observável |
| --- | --- |
| AC-01 | Depois do setup de um roteador, a pessoa abre o console, escolhe esse roteador e vê a rota padrão ativa, a porta de saída e se a checagem responde. A senha não aparece na tela nem no log. |
| AC-02 | Sem rota padrão, a tela daquele roteador mostra o estado vazio e não inventa um caminho. |
| AC-03 | Sem sessão com um roteador, a tela mostra a falha daquele agent e não mostra segredo. Os outros agents continuam visíveis. |
| AC-04 | Leases aparecem quando o DHCP daquele roteador os tem. A lista vazia é um estado válido. |
| AC-05 | Esta fatia não grava no roteador. |
| AC-06 | Dois agents cadastrados aparecem como dois roteadores. O retrato de um não se mistura com o do outro. |

### 2. Prova do USB

Confirmação, desabilitar e reabilitar a rota primária, e a medida de um destino público.

| ID | Resultado observável |
| --- | --- |
| AC-07 | Sem confirmação, a rota primária permanece ativa. |
| AC-08 | Com confirmação, a prova mede o destino pela USB e pela LAN, e a rota primária termina ativa. |
| AC-09 | Cancelar ou falhar a prova também deixa a rota primária ativa. |

### 3. Inventário de aparelhos

A pessoa abre um aparelho e vê endereço, nome e o caminho de saída associado. Nada é alterado no firewall nem na rota.

### 4. Histórico de troca de caminho

Cada troca registrada no hub tem hora e motivo. O roteador não é consultado como arquivo desse histórico.

### 5. Assistente em modo proposta

A fronteira de segredo é testada por uma lista de recusa: o retrato enviado ao modelo não contém senha, usuário, URL com credencial, export nem backup. Proposta fora da lista fechada não é executada.

### 6. Agent separado e hub remoto

O hub recebe retrato e evento. A senha não está no hub. Um comando de escrita ainda pede confirmação no agent.

## Ordem e verificação

A fatia 1 é a única que responde "por onde a casa está saindo" sem escrever no roteador. As seguintes não começam antes de a anterior estar verificada na jornada real, com um roteador de laboratório ou com confirmação explícita no roteador da casa.

Um teste automatizado cobre o contrato de cada módulo com um roteador falso. A jornada de ponta a ponta usa um RouterOS de laboratório. Teste verde sem essa jornada não fecha o aceite.

## Decisões fechadas

- A fatia 1 é uma API em Go e um frontend SvelteKit separado. A interface nasce em inglês e português do Brasil, com Paraglide. Um hub acompanha vários agents. Cada agent fala com um MikroTik.
- No setup, a pessoa escolhe o cofre. Os quatro adaptadores fazem parte dessa escolha: arquivo cifrado, Ciphera, Infisical e chaveiro do sistema.
- A leitura funciona em qualquer roteador que tenha rota padrão. Ela não exige os comentários estáveis. Esses comentários só entram na prova e no cadastro de WAN.

## Decisões em aberto

1. Endereço de checagem. Ele é configuração do tenant, nunca DNS da LAN. Falta o valor inicial sugerido na tela de setup.
2. Onde o hub remoto mora, se o modo separado for usado. Isso não muda a regra de que a senha fica no agent.
3. Qual modelo atende o assistente. Ele entra atrás do módulo de assistente, na fatia 5, e continua sem cliente do roteador.
4. Por quanto tempo guardar amostra de aparelho e de caminho.
5. Licença do repositório. Ainda não há arquivo de licença.

## O que este plano não autoriza

A leitura não aponta para um roteador de produção até haver um laboratório ou uma confirmação explícita. As decisões em aberto não bloqueiam a fatia 1.
