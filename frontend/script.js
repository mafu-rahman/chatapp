// Function to send message
function sendMessage() {
    var messageInput = document.getElementById("messageInput");
    var messageContent = messageInput.value;
    messageInput.value = ""; // Clear input field

    // Send message to backend
    fetch('http://localhost:8080/chatapp/send', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/x-www-form-urlencoded',
        },
        body: 'content=' + encodeURIComponent(messageContent),
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
    console.log(message)

    var chatMessages = document.getElementById("chatMessages");
    var messageElement = document.createElement("div");
    messageElement.classList.add("message");

    var messageContentElement = document.createElement("div");
    messageContentElement.classList.add("message-content");
    messageContentElement.textContent = message[0].content;


    var messageTimeElement = document.createElement("div");
    messageTimeElement.classList.add("message-time");
    messageTimeElement.textContent = message[0].date;

    messageElement.appendChild(messageContentElement);
    messageElement.appendChild(messageTimeElement);

    chatMessages.appendChild(messageElement);

    // Scroll to bottom
    chatMessages.scrollTop = chatMessages.scrollHeight;
}

// Display initial messages when page loads
window.onload = function() {
    receiveMessages();
};
