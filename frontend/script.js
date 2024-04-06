// Function to send message
// Define global variables to store topic name and email
var topicName = "";
var userEmail = "";

function sendMessage() {
    var messageInput = document.getElementById("messageInput");
    var senderNameInput = document.getElementById("senderName");
    var senderEmailInput = document.getElementById("senderEmail");
    var messageTopicInput = document.getElementById("messageTopic");

    var messageContent = messageInput.value;
    var senderName = senderNameInput.value;
    var senderEmail = senderEmailInput.value;
    var messageTopic = messageTopicInput.value;

    // If sender name and email are not provided, use the previous values
    if (senderName === "") {
        senderName = topicName;
    } else {
        // Update the global variable if the user changes the name
        topicName = senderName;
    }
    if (senderEmail === "") {
        senderEmail = userEmail;
    } else {
        // Update the global variable if the user changes the email
        userEmail = senderEmail;
    }

    // If topic is not provided, use the previous value
    if (messageTopic === "") {
        messageTopic = topicName;
    } else {
        // Update the global variable if the user changes the topic
        topicName = messageTopic;
    }

    messageInput.value = "";

    fetch('http://localhost:8080/chatapp/send', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/x-www-form-urlencoded',
        },
        body: 'content=' + encodeURIComponent(messageContent) +
              '&name=' + encodeURIComponent(senderName) +
              '&email=' + encodeURIComponent(senderEmail) +
              '&topic=' + encodeURIComponent(messageTopic),
    })
    .then(response => {
        if (!response.ok) {
            throw new Error('Network response was not ok');
        }
    })
    .catch(error => {
        console.error('Error sending message:', error);
    });
}

// Function to receive messages from backend using WebSocket
function receiveMessages() {
    var socket = new WebSocket('ws://localhost:8080/chatapp/websocket');

    socket.onopen = function() {
        console.log('WebSocket connection established.');
    };

    socket.onmessage = function(event) {
        var message = JSON.parse(event.data);
        displayMessage(message);
    };

    socket.onclose = function(event) {
        console.log('WebSocket connection closed:', event);
    };

    socket.onerror = function(error) {
        console.error('WebSocket error:', error);
    };
}

// Function to display message
function displayMessage(message) {
    console.log(message);

    var chatMessages = document.getElementById("chatMessages");
    var messageElement = document.createElement("div");
    messageElement.classList.add("message");

    var messageSenderElement = document.createElement("div");
    messageSenderElement.classList.add("message-sender");
    messageSenderElement.textContent = message[0].name + " (" + message[0].email + ")";

    var messageTimeElement = document.createElement("div");
    messageTimeElement.classList.add("message-time");
    messageTimeElement.textContent = message[0].date;

    var messageTopicElement = document.createElement("div");
    messageTopicElement.classList.add("message-topic");
    messageTopicElement.textContent = "Topic: " + message[0].topic;

    var messageContentElement = document.createElement("div");
    messageContentElement.classList.add("message-content");
    messageContentElement.textContent = message[0].content;

    messageElement.appendChild(messageSenderElement);
    messageElement.appendChild(messageTimeElement);
    messageElement.appendChild(messageTopicElement);
    messageElement.appendChild(messageContentElement);

    chatMessages.appendChild(messageElement);

    // Scroll to bottom
    chatMessages.scrollTop = chatMessages.scrollHeight;
}

window.onload = function() {
    receiveMessages();
};