import socket
import json
import threading
import sys
import time
import logging
import ipaddress

class DistributedRouter:
    def __init__(self, port, neighbors):
        """
        Initialize the distributed router
        
        :param port: Local port to bind
        :param neighbors: Dictionary of neighbors {address: cost}
        """
        # Network configuration
        self.port = port
        self.neighbors = neighbors
        self.routing_table = {}
        
        # Configure logging
        self.logger = self._setup_logger()
        
        # Create socket
        self.sock = self._create_socket()
        
        # Initialize routing table
        self._initialize_routing_table()
        
        # Synchronization primitives
        self.lock = threading.Lock()
        self.stop_event = threading.Event()

    def _setup_logger(self):
        """
        Configure logging for the router
        
        :return: Configured logger
        """
        logger = logging.getLogger(f'Router-{self.port}')
        logger.setLevel(logging.INFO)
        
        formatter = logging.Formatter(
            '%(asctime)s - Router - %(levelname)s - %(message)s',
            datefmt='%Y-%m-%d %H:%M:%S'
        )
        
        console_handler = logging.StreamHandler()
        console_handler.setFormatter(formatter)
        logger.addHandler(console_handler)
        
        return logger

    def _create_socket(self):
        """
        Create and configure UDP socket
        
        :return: Configured socket
        """
        try:
            sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
            sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
            sock.bind(('0.0.0.0', self.port))
            sock.settimeout(1)  # Non-blocking with short timeout
            return sock
        except Exception as e:
            print(f"Socket creation failed: {e}")
            raise

    def _initialize_routing_table(self):
        """
        Initialize routing table with neighbors
        """
        # Add self to routing table
        self.routing_table[f'localhost:{self.port}'] = {
            'cost': 0,
            'next_hop': f'localhost:{self.port}',
            'last_updated': time.time()
        }
        
        # Add neighbors
        for neighbor, cost in self.neighbors.items():
            self.routing_table[neighbor] = {
                'cost': cost,
                'next_hop': neighbor,
                'last_updated': time.time()
            }

    def bellman_ford_update(self):
        """
        Perform Bellman-Ford routing table update
        
        :return: Boolean indicating if routing table changed
        """
        updated = False
        
        with self.lock:
            for dest in list(self.routing_table.keys()):
                # Skip self
                if dest == f'localhost:{self.port}':
                    continue
                
                # Try finding better routes through neighbors
                for neighbor in self.neighbors:
                    current_cost = self.routing_table.get(dest, {}).get('cost', float('inf'))
                    
                    # Compute potential new route
                    link_cost = self.neighbors[neighbor]
                    neighbor_routes = [
                        r['cost'] for r in self.routing_table.values() 
                        if r['next_hop'] == neighbor
                    ]
                    
                    if not neighbor_routes:
                        continue
                    
                    new_cost = link_cost + min(neighbor_routes)
                    
                    # Update if new route is better
                    if new_cost < current_cost:
                        self.routing_table[dest] = {
                            'cost': new_cost,
                            'next_hop': neighbor,
                            'last_updated': time.time()
                        }
                        updated = True
        
        return updated

    def broadcast_routing_table(self):
        """
        Broadcast routing table to all neighbors
        """
        with self.lock:
            for neighbor in self.neighbors:
                try:
                    host, port = neighbor.split(':')
                    data = json.dumps(self.routing_table).encode('utf-8')
                    self.sock.sendto(data, (host, int(port)))
                except Exception as e:
                    print(f"Broadcast to {neighbor} failed: {e}")

    def receive_updates(self):
        """
        Receive and process routing updates from neighbors
        """
        while not self.stop_event.is_set():
            try:
                data, addr = self.sock.recvfrom(4096)
                received_table = json.loads(data.decode('utf-8'))
                
                with self.lock:
                    updated = False
                    for dest, route_info in received_table.items():
                        # Skip self routes
                        if dest == f'localhost:{self.port}':
                            continue
                        
                        # Update if route is new or better
                        current_route = self.routing_table.get(dest, None)
                        if (not current_route or 
                            route_info['cost'] < current_route['cost']):
                            self.routing_table[dest] = route_info
                            updated = True
                    
                    if updated:
                        self.broadcast_routing_table()
                
            except socket.timeout:
                continue
            except ConnectionResetError:
                # Ignore connection reset errors
                continue
            except Exception as e:
                print(f"Update receive error: {e}")

    def periodic_update(self):
        """
        Periodically update routing table
        """
        while not self.stop_event.is_set():
            if self.bellman_ford_update():
                self.broadcast_routing_table()
            time.sleep(5)

    def start(self):
        """
        Start router threads and CLI
        """
        try:
            # Start receive thread
            recv_thread = threading.Thread(target=self.receive_updates)
            recv_thread.daemon = True
            recv_thread.start()

            # Start periodic update thread
            update_thread = threading.Thread(target=self.periodic_update)
            update_thread.daemon = True
            update_thread.start()

            # Run CLI
            self._run_cli()
        except Exception as e:
            print(f"Router start error: {e}")
        finally:
            self.stop()

    def _run_cli(self):
        """
        Interactive command-line interface
        """
        while not self.stop_event.is_set():
            try:
                cmd = input(f"{self.port}> ").strip()
                if cmd == 'routes':
                    with self.lock:
                        print(json.dumps(self.routing_table, indent=2))
                elif cmd == 'quit':
                    self.stop_event.set()
                    break
            except (KeyboardInterrupt, EOFError):
                self.stop_event.set()
                break

    def stop(self):
        """
        Gracefully stop router
        """
        self.stop_event.set()
        self.sock.close()
        print(f"Router on port {self.port} shutting down")

def main():
    # Validate command-line arguments
    if len(sys.argv) < 4 or (len(sys.argv) - 2) % 2 != 0:
        print("Usage: python router.py <port> <neighbor1> <cost1> [neighbor2] [cost2]...")
        sys.exit(1)

    try:
        # Parse port and neighbors
        port = int(sys.argv[1])
        neighbors = {}
        
        # Validate and add neighbors
        for i in range(2, len(sys.argv), 2):
            neighbor = sys.argv[i]
            cost = float(sys.argv[i+1])
            neighbors[neighbor] = cost

        # Create and start router
        router = DistributedRouter(port, neighbors)
        router.start()
    except Exception as e:
        print(f"Router initialization error: {e}")
        sys.exit(1)

if __name__ == '__main__':
    main()