import React, { useState, useRef } from 'react';
import './App.css';

function App() {
  const [inputText, setInputText] = useState('');
  const [outputText, setOutputText] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const fileInputRef = useRef(null);

  const handleInputChange = (e) => {
    setInputText(e.target.value);
  };

  const handleClear = () => {
    setInputText('');
    setOutputText('');
  };

  const handleFileButtonClick = () => {
    fileInputRef.current.click();
  };

  const handleFileChange = (e) => {
    const file = e.target.files[0];
    if (!file) return;

    const reader = new FileReader();
    reader.onload = (event) => {
      setInputText(event.target.result);
    };
    reader.readAsText(file);
  };

  const handleAnalyze = async () => {
    if (!inputText.trim()) return;

    setIsLoading(true);
    
    try {
      const response = await fetch('http://localhost:8080/analizar', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ text: inputText }),
      });

      if (!response.ok) {
        throw new Error('Error en la respuesta del servidor');
      }

      const data = await response.json();
      
      if (data.type === 'succes') {
        setOutputText(data.message);
      } else {
        setOutputText('Error: ' + data.message);
      }
    } catch (error) {
      setOutputText('Error al conectar con el servidor: ' + error.message);
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="terminal">
      <div className="terminal-header">
        <div className="terminal-buttons">
          <span className="terminal-button close"></span>
          <span className="terminal-button minimize"></span>
          <span className="terminal-button maximize"></span>
        </div>
      </div>
      
      <div className="terminal-body">
        <input
          type="file"
          ref={fileInputRef}
          onChange={handleFileChange}
          style={{ display: 'none' }}
          accept=".txt,.json,.xml,.csv"
        />
        
        <div className="terminal-prompt">
          <span className="prompt-user">evelyn@EVA02:</span>
          <span className="prompt-path">~/proyecto1/MIA_1S2025_202112395</span>
        </div>
        
        <textarea
          className="terminal-input"
          value={inputText}
          onChange={handleInputChange}
          placeholder="Inserte Texto aqui"
          spellCheck="false"
        />
        
        <div className="terminal-buttons-container">
          <button 
            className="terminal-button-action load"
            onClick={handleFileButtonClick}
            disabled={isLoading}
          >
            {isLoading ? 'Cargando...' : 'Cargar Archivo'}
          </button>
          <button 
            className="terminal-button-action analyze"
            onClick={handleAnalyze}
            disabled={isLoading || !inputText.trim()}
          >
            {isLoading ? 'Analizando...' : 'Analizar'}
          </button>
          <button 
            className="terminal-button-action clear" 
            onClick={handleClear}
            disabled={isLoading}
          >
            Limpiar
          </button>
        </div>
        
        <div className="terminal-prompt">
          <span className="prompt-user">evelyn@EVA02:</span>
          <span className="prompt-path">~/proyecto1/MIA_1S2025_202112395</span>
        </div>
        
        <textarea
          className="terminal-output"
          value={outputText}
          readOnly
          placeholder="Resultados del análisis aparecerán aquí"
          spellCheck="false"
        />
      </div>
    </div>
  );
}

export default App;