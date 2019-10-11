# Gagarin.Network
Network expose 2 different ports 9080 for network communication between peers and 9180 for websocket communication
### Common shell commands
* generates protobuf classes
    * ```make protos```
* build gnetwork binaries
    * ```make build```
* generate mock classes
    * ```mockery --all```
* start network
    * ```gnetwork -l $i, where i=0...N```
    
### Docker
Common way to start network cluster is via Docker.
* create local network 
    * ```docker network create --driver=bridge --subnet=172.18.0.0/16 gagarin	```
* build docker image
    * ```docker build .```
* run container ($1 is node index, $2 is image short hash)
    * ```docker run --net gagarin --ip 172.18.0.1$1 -it -e GN_IND=$1 -p 918$1:9180 $2 ./gnetwork``` 

### Static dir
* peers.json
```
{
	"peers": [ #list of bootstrap peers
		{
			"addr": "0xDd9811Cfc24aB8d56036A8ecA90C7B8C75e35950", #blockchain addr
			"pub": "8d00d7cea8f1f6edc961fe2867ffb94a1e78ef8b8eb416d6f5b7b3f6dd9811cfc24ab8d56036a8eca90c7b8c75e35950", # blockchain G1 public key
			"MultiAddress": "/ip4/172.18.0.10/tcp/9080/p2p/16Uiu2HAmGgeX9Sr75ofG4rQbUhRiUH2AuKii5QCdD9h8NT83afo4" #libp2p multiaddr
		},
  ...
```
* peer(i).json, where i is the same i from start network
```
{
	"addr": "0xA60c85Be9c2b8980Dbd3F82cF8CE19DE12f3E829", #blockchain addr
	"id": "16Uiu2HAmCRVJB9iKxon89fp8J25KvEgkEmgViLnaYi7A9aiZoSpy", #libp2p peer id (public ECDSA key)
	"pk": "4a157ef4295be2413c59613cd5dd4f3b0688ac6227709a9634608f33a49250ef", #blockchain private key 
	"pkpeer": "080212208ceba2eb4883bfd2638dd37807b9986ea9ce18319b4bcf663fafb007db3087e6" #libp2p peer private ECDSA key
	"pub": "8d00d7cea8f1f6edc961fe2867ffb94a1e78ef8b8eb416d6f5b7b3f6dd9811cfc24ab8d56036a8eca90c7b8c75e35950" #blockchain pub key

}
```
* seed.json starting state
```
{
  "accounts": [ #account list
    {
      "address": "0xA60c85Be9c2b8980Dbd3F82cF8CE19DE12f3E829", #blockchain address
      "balance": 1000 #starting balance
    },
    ...
```
* settings.yaml
```
Hotstuff: #hotstuff settings 
  N: 10 #total node amount, must be 3 * n + 1, here n = 3
  Delta: 5000 #stage duration in milliseconds, 2*Delta is v iew change duration, 4* Delta is GST
  BlockDelta: 10 #max time in millisecinds to wait for transactions 
Network:
  MinPeerThreshold: 3 #minimum peer count to bootstrap connection, from bootstrap list
  ReconnectPeriod: 10000 #period after watchdog checks bootstrap connections and reconnects whether count less then MinPeerThreshold
  ConnectionTimeout: 3000 # timeout
Storage:
  Dir: /var/folders/m7/c56_pk2n0dj4xbplcndc9d140000gn/T/ #storage dir
```
##Tests
There are several tests in the project. We have very light unit tests which are placed near scripts to test and integration tests in the network/test directory.
Main integration tests are in it_test.go. Each test implements specific scenario and runs main hotstuff processes. Important: for now we mock block service and tx services and they are not covered in it_tests, tests for these services can be found in block_protocol_test.go and tx_service_test.go.

## Что такое Height, Epoch ViewNumber, Round?
### Round
Время жизни блокчейна разделено на раунды, раунд это минимальный промежуток времени по истечении которого можно говорить о гарантированном прогрессе. Каждый следующий раунд увеличивает ViewNumber на единицу. Высота создаваемого блока всегда равна номеру текущего раунда.
 
Во время раунда могут произойти следующие события:
#### Proposer
- P собрал 2*f + 1 голос и отправил остальным пирам новый блок со сформированным из голосов QC 
- P не смог собрать голоса за ∆ и отправил остальным пирам новый блок и самый старший QC известный ему
- P не смог собрать голоса и не смог отправить блок, раунд длится 2*∆ и P переходит к следующему раунда
- Р если это последний раунд в эпохе Р переходит к старту новой эпохи
#### Replica
- R ждет ∆ нового блока, когда получает его принимает решение голосовать или нет



Блокчейн растет за счет последовательного увеличения своей длинны в процессе proposing(аналогично майнингу) новых блоков. 

## Как изменить содержимое Blockchain
-  В Hello сообщении будет содержаться высота текущего блока у других пиров, мы пытаемся синхронизироваться до этой высоты и скачатьизвестные блоки. После синхрониизации высота блокчейна + 1 совпадает с текущим вью
- Когда мы в качестве репликии получаем валидный Proposal, валидным мы считаем Proposal полученный от текущего proposer
- Когда мы в качестве текущего proposer делаем Proposal
- Когда мы добавляем пустой блок и переходим к следующему вью по таймауту
- Когда мы синхронизируемся по StartEpoch и заполняем пропущенные блоки пустыми

Высота блокчейна никогда не отличается от текущего номера вью больше че на единицу . --- это верно???


## Когда могут возникнуть форки?
- Proposer послал нам блок отличный от блоков других участников, тогда

| Before | Vote | Adversary P | Vote | Proposer P |
| --- | --- | --- | --- | --- |
| B'(B) | B' | B''(B') | B'' | B'''(B'') |
| B'(B) | B' | A(B') | A | B'''(B'') | 

У второго пира будет форк B - B' - A B - B' - B'' - B''', даже если второй и последующий Proposer окажутся злоумышленниками, 
в худшем случае они смогут растить форки на блоке B' подобные А, очевидно, что они не соберут QC для своего блока и второй пир рано или поздно узнает новый QC
Более того, в этом случае у нас будет доказательство мошенничества в виде блока B'' из соответствующего QC и блока A из proposal

- Adversary может не послать блок и тогда он либо не достигнет прогресса и мы получим новый блок на этой высоте от нового Proposer,
в случае если он не собрал QC, что аналогично первому случаю для пииров который получили Proposal. 
Либо Proposer соберет нужное количество воутов и соберет QС, который мы получим в новом блоке, что аналогично проблеме в сети.

## Equivocations
### Как работают PoE 
Proofs of equivocation (PoE) не являются частью протокола консенсуса, а будут перенесен на следующий уровень, в протокол который будет реализовывать финансовый уровень
Таким образом PoE будут специальными транзакцииями, по типу майнинговых, где penalty будет переводиться нашедшему нарушение пропоузеру. 
Комиссию при этом будет платить пропоузер, который включил poe в блок. В случае невалидного poe комиссия будет просто списваться с нашедшего без зачисления ему награды

### Equivocation
Vote
- Голосование за несколько блоков на одной высоте одному пропозеру (легкий пруф)
- Голосование за несколько блоков на одной высоте разным пропоузерам (такого не может быть) (легкий пруф)
- Голосование за блок, который не экстендит PREF() (форки) (легкий пруф прекоммит серт у всех один)
- Голосование за невалидный блок (легкий пруф)
- Голосование другому пропозеру (вряли нарушение)
- Не проголосовать ни за что (Withheld предыдущий пропозер мог не прислать пропозал никому, что не возможно доказать)

Proposal
- Пропозал блока который не экстендит PREF() (легкий пруф прекоммит серт у всех один)
- пропозал не на своей высоте (легкий пруф)
- пропозал не всем (Withheld не доказуемо, здесь можно поиграть с экономической мотивацией)
- пропозал невалидного блока (легкий пруф)
- Пропозал разных блоков разным пирам (форки если прогресс есть, то кто-то увидит блок за который он голосовал в отказаном форке и у него будет доказательство вины пропозера)

QC
- Невалидный QC (легкий пруф)
- QC на изъятый блок -- не возможно, этот блок видели как минимум f + 1 честных пиров
- QC лежит на другом форке (легкий пруф)
StartEpoch 
неверная подпись сообщения
неверная высота

Withheld
самое сложное доказать withheld, очевидное решение это давать награду за воут, которая будет превышать награду за ложный донос, но тут можно не учесть все факторы и выгода от ложного доноса может перевесить выгоду от голосования.

### Ситуации в которых не удается доказать Equivocations
-Eclipse attack. Proposal withheld направленный, который не останавливает прогресс основной части сети
Злоумышленники могут действовать в сговоре и заставить конкретного пира отстать от основной сети и пропустить свою очередь делать proposal или экстендить не PREF() форк.
Первое возможно, если злоумышленники не посылаюст proposals намеренно, второе возможно, если когда злоумышленники заставляют жертву делать пропозал без части блоков (онии их изъяли), в таком случае жертва заполняет пробелы пустыми блоками, расщиряя неосновную ветку
В общем случае мы не можем наказывать за пропуск своей очереди для пропозинга, это проблема
