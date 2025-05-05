import React, { useState, useEffect } from 'react';

const PartitionSelector = ({ disk, onSelect }) => {
  const [partitions, setPartitions] = useState([]);

  useEffect(() => {
    const fetchPartitions = async () => {
      const response = await fetch('http://localhost:8080/api/command', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ input: `fdisk -path=${disk.path}` }),
      });
      const data = await response.json();

      // Simulación simple: cada línea = partición
      const lines = data.result.split('\n').filter(Boolean);
      const parsed = lines.map((line, i) => {
        const [name, size, fit, status] = line.split(',');
        return { id: i, name, size, fit, status };
      });
      setPartitions(parsed);
    };
    fetchPartitions();
  }, [disk]);

  const handleMount = async (partition) => {
    const input = `mount -path=${disk.path} -name=${partition.name}`;
    await fetch('http://localhost:8080/api/command', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ input }),
    });
    onSelect(partition); // Ir al explorador
  };

  return (
    <div>
      <h2>Particiones en {disk.name}</h2>
      <ul>
        {partitions.map((p) => (
          <li key={p.id}>
            <button onClick={() => handleMount(p)}>
              {p.name} — {p.size} — {p.status}
            </button>
          </li>
        ))}
      </ul>
    </div>
  );
};

export default PartitionSelector;
