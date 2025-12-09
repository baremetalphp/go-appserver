<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Welcome to Bare Metal PHP</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Roboto', 'Oxygen', 'Ubuntu', 'Cantarell', 'Fira Sans', 'Droid Sans', 'Helvetica Neue', sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
            color: #333;
        }

        .container {
            background: white;
            border-radius: 20px;
            box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
            max-width: 900px;
            width: 100%;
            overflow: hidden;
        }

        .header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 60px 40px;
            text-align: center;
        }

        .header h1 {
            font-size: 3.5rem;
            font-weight: 800;
            margin-bottom: 10px;
            letter-spacing: -1px;
        }

        .header .subtitle {
            font-size: 1.3rem;
            opacity: 0.95;
            font-weight: 300;
        }

        .content {
            padding: 50px 40px;
        }

        .badge {
            display: inline-block;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 8px 20px;
            border-radius: 50px;
            font-size: 0.85rem;
            font-weight: 600;
            margin-bottom: 30px;
            letter-spacing: 0.5px;
        }

        .features {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 25px;
            margin: 40px 0;
        }

        .feature {
            background: #f8f9fa;
            padding: 30px;
            border-radius: 12px;
            border: 2px solid #e9ecef;
            transition: all 0.3s ease;
        }

        .feature:hover {
            border-color: #667eea;
            transform: translateY(-5px);
            box-shadow: 0 10px 25px rgba(102, 126, 234, 0.15);
        }

        .feature-icon {
            font-size: 2.5rem;
            margin-bottom: 15px;
        }

        .feature h3 {
            font-size: 1.3rem;
            margin-bottom: 10px;
            color: #2d3748;
        }

        .feature p {
            color: #718096;
            line-height: 1.6;
        }

        .quick-start {
            background: #f8f9fa;
            border-radius: 12px;
            padding: 30px;
            margin-top: 40px;
            border-left: 4px solid #667eea;
        }

        .quick-start h2 {
            color: #2d3748;
            margin-bottom: 20px;
            font-size: 1.5rem;
        }

        .code-block {
            background: #2d3748;
            color: #e2e8f0;
            padding: 20px;
            border-radius: 8px;
            font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
            font-size: 0.9rem;
            line-height: 1.8;
            overflow-x: auto;
            margin: 15px 0;
        }

        .code-block code {
            color: #68d391;
        }

        .code-block .comment {
            color: #a0aec0;
        }

        .footer {
            text-align: center;
            padding: 30px;
            color: #718096;
            border-top: 1px solid #e9ecef;
        }

        .footer a {
            color: #667eea;
            text-decoration: none;
            font-weight: 600;
        }

        .footer a:hover {
            text-decoration: underline;
        }

        @media (max-width: 768px) {
            .header h1 {
                font-size: 2.5rem;
            }

            .header .subtitle {
                font-size: 1.1rem;
            }

            .content {
                padding: 30px 20px;
            }

            .features {
                grid-template-columns: 1fr;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🚀 Bare Metal PHP</h1>
            <p class="subtitle">A lightweight, educational PHP framework</p>
        </div>

        <div class="content">
            <span class="badge">✨ Fresh Installation</span>

            <p style="font-size: 1.1rem; color: #4a5568; line-height: 1.8; margin-bottom: 30px;">
                Welcome! You've successfully installed Bare Metal PHP. This is a minimal, educational framework
                designed to help you understand how modern PHP frameworks work under the hood.
            </p>

            <div class="features">
                <div class="feature">
                    <div class="feature-icon">🎯</div>
                    <h3>Routing</h3>
                    <p>Simple, expressive routing system with support for RESTful routes and middleware.</p>
                </div>

                <div class="feature">
                    <div class="feature-icon">💾</div>
                    <h3>Database ORM</h3>
                    <p>ActiveRecord-style ORM with migrations, relationships, and query builder.</p>
                </div>

                <div class="feature">
                    <div class="feature-icon">🎨</div>
                    <h3>Views</h3>
                    <p>Template engine with layouts, components, and clean PHP syntax.</p>
                </div>

                <div class="feature">
                    <div class="feature-icon">🔧</div>
                    <h3>Service Container</h3>
                    <p>Dependency injection container for managing application services.</p>
                </div>

                <div class="feature">
                    <div class="feature-icon">📦</div>
                    <h3>Migrations</h3>
                    <p>Database schema versioning with rollback support and multiple drivers.</p>
                </div>

                <div class="feature">
                    <div class="feature-icon">⚡</div>
                    <h3>Fast & Light</h3>
                    <p>Minimal overhead, no unnecessary dependencies, built for learning.</p>
                </div>
            </div>

            <div class="quick-start">
                <h2>🚀 Quick Start</h2>

                <p style="margin-bottom: 15px; color: #4a5568;"><strong>1. Run migrations:</strong></p>
                <div class="code-block">
<span class="comment"># Create your database tables</span>
php mini migrate
                </div>

                <p style="margin-bottom: 15px; color: #4a5568; margin-top: 25px;"><strong>2. Start the development server:</strong></p>
                <div class="code-block">
<span class="comment"># Start the built-in PHP server</span>
php mini serve
                </div>

                <p style="margin-bottom: 15px; color: #4a5568; margin-top: 25px;"><strong>3. Create your first route:</strong></p>
                <div class="code-block">
<span class="comment"># Edit routes/web.php</span>
$router->get('/hello', function () {
    return new Response('Hello World!');
});
                </div>
            </div>
        </div>

        <div class="footer">
            <p>Built with ❤️ for learning | <a href="https://github.com" target="_blank">Documentation</a> | <a href="https://github.com" target="_blank">GitHub</a></p>
        </div>
    </div>
</body>
</html>
