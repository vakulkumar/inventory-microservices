import React, { useState, useEffect } from 'react';
import Navbar from './components/Navbar';
import ProductCard from './components/ProductCard';
import Cart from './components/Cart';
import { productService, orderService } from './services/api';

function App() {
    const [products, setProducts] = useState([]);
    const [cart, setCart] = useState([]);
    const [loading, setLoading] = useState(true);
    const [isCartOpen, setIsCartOpen] = useState(false);
    const [isCheckingOut, setIsCheckingOut] = useState(false);
    const [successMessage, setSuccessMessage] = useState('');

    useEffect(() => {
        loadProducts();
    }, []);

    const loadProducts = async () => {
        try {
            setLoading(true);
            const data = await productService.getAllProducts();
            setProducts(data || []);
        } catch (error) {
            console.error('Error loading products:', error);
            // Use mock data if API fails
            setProducts([
                {
                    id: 1,
                    name: 'Premium Wireless Headphones',
                    description: 'High-quality audio with active noise cancellation',
                    price: 299.99,
                    stock: 25
                },
                {
                    id: 2,
                    name: 'Smart Watch Pro',
                    description: 'Track your fitness and stay connected',
                    price: 399.99,
                    stock: 15
                },
                {
                    id: 3,
                    name: 'Mechanical Keyboard',
                    description: 'RGB backlit with mechanical switches',
                    price: 149.99,
                    stock: 8
                },
                {
                    id: 4,
                    name: 'Wireless Mouse',
                    description: 'Ergonomic design with precision tracking',
                    price: 79.99,
                    stock: 50
                },
                {
                    id: 5,
                    name: '4K Monitor',
                    description: '27-inch ultra HD display',
                    price: 499.99,
                    stock: 12
                },
                {
                    id: 6,
                    name: 'USB-C Hub',
                    description: 'Multi-port adapter for all your devices',
                    price: 59.99,
                    stock: 30
                }
            ]);
        } finally {
            setLoading(false);
        }
    };

    const handleAddToCart = (product) => {
        const existingItem = cart.find(item => item.id === product.id);

        if (existingItem) {
            setCart(cart.map(item =>
                item.id === product.id
                    ? { ...item, quantity: item.quantity + 1 }
                    : item
            ));
        } else {
            setCart([...cart, { ...product, quantity: 1 }]);
        }

        showSuccessMessage('Added to cart! 🎉');
    };

    const handleUpdateQuantity = (productId, newQuantity) => {
        if (newQuantity < 1) return;

        setCart(cart.map(item =>
            item.id === productId
                ? { ...item, quantity: newQuantity }
                : item
        ));
    };

    const handleRemoveItem = (productId) => {
        setCart(cart.filter(item => item.id !== productId));
    };

    const handleCheckout = async () => {
        if (cart.length === 0) return;

        setIsCheckingOut(true);

        try {
            // Place orders for each item
            const orderPromises = cart.map(item =>
                orderService.createOrder({
                    product_id: item.id,
                    quantity: item.quantity
                })
            );

            await Promise.all(orderPromises);

            // Clear cart and show success
            setCart([]);
            setIsCartOpen(false);
            showSuccessMessage('Order placed successfully! 🎊');

            // Reload products to get updated stock
            loadProducts();
        } catch (error) {
            console.error('Error placing order:', error);
            alert('Failed to place order. Please try again.');
        } finally {
            setIsCheckingOut(false);
        }
    };

    const showSuccessMessage = (message) => {
        setSuccessMessage(message);
        setTimeout(() => setSuccessMessage(''), 3000);
    };

    const cartItemCount = cart.reduce((sum, item) => sum + item.quantity, 0);

    return (
        <div className="app">
            <Navbar
                cartCount={cartItemCount}
                onCartClick={() => setIsCartOpen(true)}
            />

            <div className="container">
                {/* Hero Section */}
                <div className="hero">
                    <div className="hero-content">
                        <h1>Welcome to ShopHub</h1>
                        <p>Discover amazing products at unbeatable prices. Your one-stop shop for everything you need!</p>
                    </div>
                </div>

                {/* Success Message */}
                {successMessage && (
                    <div className="success-message">
                        {successMessage}
                    </div>
                )}

                {/* Products Section */}
                <section className="products-section">
                    <div className="section-header">
                        <h2 className="section-title">Featured Products</h2>
                    </div>

                    {loading ? (
                        <div className="loading">
                            <div className="spinner"></div>
                            <p>Loading amazing products...</p>
                        </div>
                    ) : (
                        <div className="products-grid">
                            {products.map(product => (
                                <ProductCard
                                    key={product.id}
                                    product={product}
                                    onAddToCart={handleAddToCart}
                                />
                            ))}
                        </div>
                    )}
                </section>
            </div>

            {/* Cart Modal */}
            {isCartOpen && (
                <Cart
                    cart={cart}
                    onClose={() => setIsCartOpen(false)}
                    onUpdateQuantity={handleUpdateQuantity}
                    onRemoveItem={handleRemoveItem}
                    onCheckout={handleCheckout}
                    isCheckingOut={isCheckingOut}
                />
            )}
        </div>
    );
}

export default App;
