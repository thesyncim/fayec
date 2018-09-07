var http = require('http'),
    faye = require('faye');

var server = http.createServer(),
    bayeux = new faye.NodeAdapter({mount: '/faye', timeout: 45});

var unauthorized = [
    '/unauthorized',
];

bayeux.addExtension({
    incoming: function (message, callback) {
        if (message.channel === '/meta/subscribe') {
            if (unauthorized.indexOf(message.subscription) >= 0) {
                message.error = '500::unauthorized channel';
            }
        }
        callback(message);
    }
});

bayeux.attach(server);
server.listen(8000);