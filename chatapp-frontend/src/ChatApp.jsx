import React, { useState, useEffect, useRef} from 'react';
import './styles.css';


function ChatApp() {
    const [messages, setMessages] = useState([]);
    const [messageInput, setMessageInput] = useState('');
    const [senderName, setSenderName] = useState('');
    const [senderEmail, setSenderEmail] = useState('');
    const [messageTopic, setMessageTopic] = useState('');
    const chatMessagesRef = useRef(null);

  useEffect(() => {
    viewChatHistory();
    receiveMessages();
  }, []);

  useEffect(() => {
    chatMessagesRef.current.scrollTop = chatMessagesRef.current.scrollHeight;
  }, [messages]);

  function sendMessage() {
    var messageContent = messageInput;
    var name = senderName;
    var email = senderEmail;
    var topic = messageTopic;

    fetch('http://localhost:8080/chatapp/send', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded',
      },
      body: 'content=' + encodeURIComponent(messageContent) +
            '&name=' + encodeURIComponent(name) +
            '&email=' + encodeURIComponent(email) +
            '&topic=' + encodeURIComponent(topic),
    })
    .then(response => {
      if (!response.ok) {
        throw new Error('Network response was not ok');
      }
    })
    .catch(error => {
      console.error('Error sending message:', error);
    });

    setMessageInput('');
  }

  function receiveMessages() {
    var socket = new WebSocket('ws://localhost:8080/chatapp/websocket');

    socket.onopen = function() {
      console.log('WebSocket connection established.');
    };

    socket.onmessage = function(event) {
      var message = JSON.parse(event.data);
      displayMessage(message[0]);
    };

    socket.onclose = function(event) {
      console.log('WebSocket connection closed:', event);
    };

    socket.onerror = function(error) {
      console.error('WebSocket error:', error);
    };
  }

  function viewChatHistory() {
    fetch('http://localhost:8080/chatapp/history')
      .then(response => {
        if (!response.ok) {
          throw new Error('Failed to fetch chat history');
        }
        return response.json();
      })
      .then(messages => {
        setMessages(messages);
      })
      .catch(error => {
        console.error('Error fetching chat history:', error);
      });
  }

  function displayMessage(message) {
    setMessages(prevMessages => {
      // Checking if prevMessages is null or undefined
      if (!prevMessages) {
        prevMessages = [];
      }
      // Spread prevMessages and append the new message
      return [...prevMessages, message];
    });
  }

  return (
    <div className="chat-container">
      <div className="chat-messages" ref={chatMessagesRef}>
        {messages && messages.map((message, index) => (
          <div key={index} className="message">
            <div className="message-sender">
              {message.name} ({message.email})
            </div>
            <div className="message-time">{message.date}</div>
            <div className="message-topic">Topic: {message.topic}</div>
            <div className="message-content">{message.content}</div>
          </div>
        ))}
      </div>
      <div className="chat-input">
        <input
          type="text"
          className="message-input"
          placeholder="Type your message..."
          value={messageInput}
          onChange={(e) => setMessageInput(e.target.value)}
        />
        <input
          type="text"
          className="sender-name"
          placeholder="Your Name"
          value={senderName}
          onChange={(e) => setSenderName(e.target.value)}
        />
        <input
          type="text"
          className="sender-email"
          placeholder="Your Email"
          value={senderEmail}
          onChange={(e) => setSenderEmail(e.target.value)}
        />
        <input
          type="text"
          className="message-topic-input"
          placeholder="Message Topic"
          value={messageTopic}
          onChange={(e) => setMessageTopic(e.target.value)}
        />
        <button className="send-button" onClick={sendMessage}>
          Send
        </button>
      </div>
    </div>
  );
}

export default ChatApp;
