import React, { useEffect, useState } from 'react';

const DiskSelector = ({ onSelect }) => {
  const [disks, setDisks] = useState([]);

  useEffect(() => {
    const fetchDisks = async () => {
      const response = await fetch('http://localhost:8080/api/command', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ input: 'mounted' }),
      });
      const data = await response.json();
      const lines = data.result.split('\n').filter(Boolean);
      const parsed = lines.map((line, i) => {
        const [path, name] = line.split(',');
        return { id: i, name: name.trim(), path: path.trim() };
      });
      setDisks(parsed);
    };
    fetchDisks();
  }, []);

  return (
    <div>
      <h2>Selecciona un Disco</h2>
      <ul>
        {disks.map((disk) => (
          <li key={disk.id}>
            <button onClick={() => onSelect(disk)}>{disk.name} — {disk.path}</button>
          </li>
        ))}
      </ul>
    </div>
  );
};

export default DiskSelector;
