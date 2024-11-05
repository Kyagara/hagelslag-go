package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
	"unsafe"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type HTTP struct{}

func (s HTTP) Name() string {
	return "http"
}

func (s HTTP) Network() string {
	return "tcp"
}

func (s HTTP) Port() string {
	return "80"
}

func (s HTTP) Scan(ip string, conn net.Conn) ([]byte, int64, error) {
	request := []string{"GET / HTTP/1.1\r\nHost: ", ip, "\r\nConnection: close\r\n\r\n"}
	get := strings.Join(request, "")

	start := time.Now()
	_, err := conn.Write(unsafe.Slice(unsafe.StringData(get), len(get)))
	if err != nil {
		return nil, 0, fmt.Errorf("sending GET request: %w", err)
	}

	response := make([]byte, 17)

	_, err = io.ReadFull(conn, response)
	if err != nil {
		return nil, 0, fmt.Errorf("reading status from GET response: %w", err)
	}

	latency := time.Since(start).Milliseconds()

	// Check if the status code is 2xx.
	if response[9] != '2' {
		return nil, 0, nil
	}

	response, err = io.ReadAll(conn)
	if err != nil {
		return nil, 0, fmt.Errorf("reading GET response: %w", err)
	}

	return response, latency, nil
}

func (s HTTP) Save(ip string, latency int64, data []byte, collection *mongo.Collection) error {
	document := bson.M{
		"_id":     ip,
		"latency": latency,
		"data":    *(*string)(unsafe.Pointer(&data)),
	}

	filter := bson.M{"_id": ip}
	opts := options.Replace().SetUpsert(true)

	_, err := collection.ReplaceOne(context.TODO(), filter, document, opts)
	if err != nil {
		return err
	}

	return nil
}

// ''''...................                                                      ..              .,<,^^^"<]l^"l?l"^I]<"^^^,>,.
// '''....................                                                     ...              .,>,^^^"<]I^^;_I^^;?>"``^,>,.
// '''.......................                                                                   .,i,^``">?;^^;_I^^;?>"```,i,.
// ''''.......................                                              ..                  .,i,```^i?;`^;+;^`;->^```"i,.
// '''''''.....................                           .'................  ........          .,i"```^i-:``:+;``:-i^```"!,.
// '''''''......................                      .''..                           ....      .,!"```^!_:``:~;``:-!^```"!,.
// '''''''.......................                   ''.                                   ...   .,!,```^l-:``:~:``:_!^```"!,.
// '''''''''.......................               ''        ........       ...'.....        ...'`:;,,^'`;-,``:~:``:_!````"l,.
// '''''''''''.........................''''''' .`.       ...                   ...  ...       .^'     ...`^``,<:``,+l`''`"l,.
// '''''''''''......................`"`......'"`       ..  ...                    ...  ..       .'        '`",<:``,+l`'''^l,.
// ''''''''''''''.................''     .. .'        ` .''                         .'   '.       `'...     '_<,`',+I`'''^l,.
// """"""""""""""""""^^^^^^^^^^^"`   ....  ".       .'.'.               .             .'  .'       .'  .'    .-,'',~I`'''^I,.
// 11111111{{11111111111111{{{}}'   '.    `        .''.             '   i'  .           '.  '        `   '.   .;'',~I`'''^I,.
// ||\\\|\|||||||||||\\\|||||(|^  .'    ''        '^'              '.  :``.  `           .'  '        `   '.   ^^',+;`'''^I,.
// {{{{{{}}}}}{{}}}}}}{{}}}}}[+  .`    ^"        ',.              '. .,'. .'  `            '  `        '   `    i":~;`''`:_:.
// <<>>>>>>>>>>>>>iiiiiiiii!!_.  `    `^        ."               `  '^.     '  '            '. `        `   `   `++?I`'`;1|;.
// ??+~~~~~~~~~~~~~<<<<<<<<>>:  ^    ^^         `              '' '`.        .' .'           '. '        `  .'   ?(|l`^i?_+,.
// +~illllllllllllIIIIIIII;;~. ..   ``.        ..          ..`"'`^.            .'."`...       '.`        .'  `   ,!}+!++,"I,.            '!
// ~<lII;;IIIIII;;;;;;;;;;;:;  `   ''`         `       .... ''."^`.            .^^``'.....     `.'        `   '  ``!)1>^'`;,.          'in8
// <>l;;;;;;;;;;;;;;;;;;;;:;`  `  .``          `       ...,""^'                    '`:,'.'.     `^         '  `   I<-:`''`;,.         .I#B$
// >>I;;;;;;;;;;::::::;;;::+  `   `.'         `         ','.                           ''.      .l         `  '.  +<"'..'">,.          ^x$$
// ii;::::::::::::::;;:::::I  ^   '^          `       '^'                                 ...    `         ..  `  ,`'..'^_(;.          .<&$
// i!;:::::::::::::::::::::,  ^  ^ ^          `    ...                                       ....`          `  `  '`..'^+/|;.           `|8
// i!::::::::::::::::::::::^  `  ^'.          `..''                              ....            `          ^  ^   ,.'^~{?+,.            ,v
// !l:::,:::::::::::::::::l'  ` .'`           `      ......''``.                 `:,^`'....      ^          `  `   ^'^~}!,I,.            ._
// !l:::,,::::::::::::::,:~`.'`.`.,  .      ` `     `!<?)fz8$$@+                :&$$$$$Bzf;`     `          "..'...""_[l`^;,.             `
// !l:,,,,,,:::Il;,"^``''.        `  :      ^ `    ;$$$$$$$$$8x;                 u@$$$$$$$$B"    `       `  `         .''`,,'.
// !l;II:,"^`''       ...''````^^",  _      '.`     I/j1~:^'.                       .'`^:!?]'    `       `  <""^^``''.       ..........
// +".  ...'""^^^"""~,"^~`````````;  ^.      `.'                                                '. '     : ':````````^_"^^,``''''.'..  ...`
// ll;"`'.   '``^````:  .^````````;  '`      '.^                                                ` .'    ', `:````````"".`;```^,^`'    .....
// lI:,,,,;:"^`'  .'..    '^"`````^^ i`'      ,.`                                               ` `     `^'^,`````````  ```^'.   ....'
// lI:,,,,,,;+t#8x}l"'     ..`````^I :``.     .,`.                                             ` '.    ;^`l^``````"^'.      .....
// !l:,,,,:]zB@@@@@BB%8c('...      :"'+I.'     `.,.                                           .'`"    `.:_;:^^^``^'.  .`^"i;.
// }_I:,,>xB$$$$$$$$@@@%c  . ."...`?_^I-.'".    ' ^.                                         ."'"   .` .(lli   ......-!_I:<:.
// $BMn||#$$$$$$$$$$$$$$?  '  ^   ',"<,+^^`,"   '.                 .    '.   ..             .".`.  `:,:<i,^^...^     Ii-!;<;.
// $$$$$$$$$$$$$$$$$$$$@"  `  ^    !^r~:;``,l,`' `.                 ....  ....                '. `,^```^`|     '     "i?!;~;'           .'"
// $$$$$$$$$$$$$$$$@$$$8   ^  `    ;"W/^```I:"",,.`.                                        .`'';``````^;W.   '.     '+?iI+;'      ..`,~\cW
// $$$$$$$$$$$$$$$$$$$&x,'.^  ^    ,l&*^```i,""","                                         "`,' ;``````')W'   `       }]>l_I'  .'^I[fzW8888
// $$$$$$$$$$$$$$$$$$$$$@&unI^"    ^}W8^```i"""""",`                                       ''   I``:``` #&'   `       _[~i]+,:+|u#&&&888888
// $$$$$$$$$$$$$$$$$$$$$$$$$$%z?^.  v/&,```!"""""""",`.                                  ''.    l``!``''#u'   "    .  >|(fz##&8&&8888888888
// $$$$$$$$$$$$$$$$$$$$$$$$$$$$$%rl'(|M^```l,""""""""","'                             .''      .l``!`` `xr'  ``    ,`IxW&888888888888888888
// $$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$@x~\!^```;:"""""""""""",^`.                      '::.        .;`";`. .]~  .I..`;}v#&8&8888888888888888888
// $$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$v)"````,l""""""""""""""!^"``'              '`^,u$j'        ',`i^'   `  .:1]r*&&&&88888888888888888888%%
// $$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$*|"```^<""""""""""i}(/ujfjrx#v|~:"^^,I_|jj{[?_*$#^.       `"^<`    `."(*&&8&8888888888888888888888%%%%
// $$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$8-^```<^^^^^^"";l|/\uz**zzzzzcccccccczz**##MMnru#i.      "^"!.   '1f#8&&88888888888888888888888%%%%%%
// $$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$zi^``~"^^^^^^^-;II;[;+1|[??_+~>>~~_-?[}1{{_!I;:::l      ,`>'  '>x&&&&888888888888888888888%8%%%%%%%%
// $$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$r;^`>^^^^^^^^l.   I::;' `;+?!^''''....       .'^`      ;^; '>x&&&8888888888888888888888%%88%%%%%%%%
// $$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$B("^>^^^^^^^^,    ;,:<."+#&&#_'                 ^      I;."/W&888888888888888888888888%%%%%%%%%%%%B
// $$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$W+^;,^^^^^^^:    ,,:<.i}&WW&]^                 `     .>!<v888888888888888888888888%%%%%%%%%%%%%%BB
// $$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$n;^!^^^^^",i.   :,:i..^<\/<`                 .^. .. `1)M88888888888888888888%%%%%%%%%%%%%%%%%%BBB
